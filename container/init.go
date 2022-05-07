package container

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/sirupsen/logrus"
)

func readUserCommand() []string {
	pipe := os.NewFile(uintptr(3), "pipe")
	msg, err := ioutil.ReadAll(pipe)
	if err != nil {
		logrus.Errorf("init read pipe error %v", err)
		return nil
	}
	msgStr := string(msg)
	return strings.Split(msgStr, " ")
}

func RunContainerInitProcess() error {
	cmdArr := readUserCommand()
	if cmdArr == nil || len(cmdArr) == 0 {
		return fmt.Errorf("Run container get user command error, cmdArr is nil")
	}

	setUpMount()

	path, err := exec.LookPath(cmdArr[0])
	if err != nil {
		logrus.Errorf("Exec loop path error %v", err)
		return err
	}
	logrus.Infof("Find path %s", path)
	if err := syscall.Exec(path, cmdArr[0:], os.Environ()); err != nil {
		logrus.Errorf(err.Error())
	}
	return nil
}

// mount init
func setUpMount() {
	// get current path
	pwd, err := os.Getwd()
	if err != nil {
		logrus.Errorf("Get current location error %v", err)
		return
	}
	logrus.Infof("current location is %s", pwd)
	pivotRoot(pwd)

	// mount proc
	// systemd 加入linux后 mount namespace 需要变成 shared by default
	// 所以必须显式声明要这个新的mount namespace 独立
	syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, "")
	defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV
	syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), "")

	syscall.Mount("tmpfs", "/dev", "tmpfs", syscall.MS_NOSUID|syscall.MS_STRICTATIME, "mode=755")
}

func isExists(path string) bool {
  _, err := os.Stat(path)
  return os.IsExist(err)
}

func pivotRoot(root string) error {
	if err := syscall.Mount(root, root, "bind", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("Mount rootfs to itself error: %v", err)
	}
	// create rootfs /.privot_root, save old_root
	pivotDir := filepath.Join(root, ".pivot_root")
	if !isExists(pivotDir) {
		if err := os.Mkdir(pivotDir, 0777); err != nil {
			return err
		}
	}
  logrus.Infof("root:%s, pivotRoot:%s", root, pivotDir)
	// privot_root mount to new rootfs, old_root mount rootfs/.privot_root
	if err := syscall.PivotRoot(root, pivotDir); err != nil {
    logrus.Errorf("pivotRoot Error: %v",err)
		return fmt.Errorf("pivot_root %v", err)
	}
	// modify the workspace to root dir
	if err := syscall.Chdir("/"); err != nil {
		return fmt.Errorf("Chdir / %v", err)
	}

	pivotDir = filepath.Join("/", ".privot_root")
	// umount rootfs/.pivot_root
	if err := syscall.Unmount(pivotDir, syscall.MNT_DETACH); err != nil {
		return fmt.Errorf("unmount pivot_root dir %v", err)
	}
	// delete temp file dir
	return os.Remove(pivotDir)
}
