package container

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

  "minidocker/utils"
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
    os.Exit(1)
	}
	logrus.Infof("current location is %s", pwd)
  if err := pivotRoot(pwd); err != nil {
    logrus.Errorf("pivotRoot exec failed! %v", err)
    os.Exit(1)
  }

	// mount proc
	// systemd 加入linux后 mount namespace 需要变成 shared by default
	// 所以必须显式声明要这个新的mount namespace 独立
  if err := syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""); err != nil {
    logrus.Errorf("mount / failed! %v", err)
    os.Exit(1)
  }
	defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV
  if err := syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), ""); err != nil {
    logrus.Errorf("mount /proc failed! %v", err)
    os.Exit(1)
  }

  if err := syscall.Mount("tmpfs", "/dev", "tmpfs", syscall.MS_NOSUID|syscall.MS_STRICTATIME, "mode=755"); err != nil {
    logrus.Errorf("mount /dev failed! %v", err)
    os.Exit(1)
  }

  logrus.Infof("success")
}

func pivotRoot(newRootDir string) error {
	if err := syscall.Mount(newRootDir, newRootDir, "bind", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("Mount rootfs to itself error: %v", err)
	}
	// create rootfs /.old_root, save old_root
	oldRootDir := filepath.Join(newRootDir, ".old_root")
	if !utils.PathExists(oldRootDir) {
		if err := os.Mkdir(oldRootDir, 0777); err != nil {
			return err
		}
	}
	// privot_root mount to new rootfs, old_root mount rootfs/.privot_root
  logrus.Infof("new: %s, old: %s", newRootDir, oldRootDir)
	if err := syscall.PivotRoot(newRootDir, oldRootDir); err != nil {
    logrus.Errorf("pivotRoot Error: %v",err)
		return fmt.Errorf("pivot_root %v", err)
	}
	// modify the workspace to root dir
	if err := syscall.Chdir("/"); err != nil {
		return fmt.Errorf("Chdir / %v", err)
	}

	oldRootDir = filepath.Join("/", ".old_root")
	// umount rootfs/.pivot_root
	if err := syscall.Unmount(oldRootDir, syscall.MNT_DETACH); err != nil {
		return fmt.Errorf("unmount old_root %s dir %v", oldRootDir, err)
	}
	// delete temp file dir
	return os.Remove(oldRootDir)
}
