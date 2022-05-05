package container

import (
	"os"
	"os/exec"
	"syscall"

	"github.com/sirupsen/logrus"
)

func NewPipe() (*os.File, *os.File, error) {
	read, write, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}
	return read, write, err
}

func NewParentProcess(tty bool) (*exec.Cmd, *os.File) {
	readPipe, writePipe, err := NewPipe()
	if err != nil {
		logrus.Errorf("New pipe error %v", err)
	}
	cmd := exec.Command("/proc/self/exe", "init")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNS | syscall.CLONE_NEWNET | syscall.CLONE_NEWIPC,
	}
	if tty {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	cmd.ExtraFiles = []*os.File{readPipe}
	mntURL := "/root/mnt/"
	rootURL := "/root/"
  NewWorkSpace(rootURL, mntURL)
	cmd.Dir = mntURL
	return cmd, writePipe
}

func PathExists(path string) (bool, error) {
  _, err := os.Stat(path)
  if err == nil {
    return true, nil
  }
  if os.IsExist(err) {
    return false, err
  }
  return false, err
}

func NewWorkSpace(rootURL string, mntURL string) {
  CreateReadOnlyLayer(rootURL)
  CreateWriteLayer(rootURL)
  CreateMountPoint(rootURL, mntURL)
}

func CreateReadOnlyLayer(rootURL string) {
  busyboxURL := rootURL + "busybox/"
  busyboxTarURL := rootURL + "busybox.tar"

  exist, err := PathExists(busyboxURL)
  if err != nil {
    logrus.Infof("Failed to judge whetjer dir %s exists. %v", busyboxURL, err)
  }
  if exist == false {
    if err := os.Mkdir(busyboxURL, 0777); err != nil {
      logrus.Errorf("Mkdir dir %s error. %v", busyboxURL, err)
    }
    if _, err := exec.Command("tar", "-xvf", busyboxTarURL, "-C", busyboxURL).CombinedOutput(); err != nil {
      logrus.Errorf("Untar dir %s, error %v", busyboxTarURL, err)
    }

  }
}

func CreateWriteLayer(rootURL string) {
  writeURL := rootURL + "writelayer/"
  if err := os.Mkdir(writeURL, 0777); err != nil {
    logrus.Errorf("Mkdir dir %s error %v", writeURL, err)
  }
}

func CreateMountPoint(rootURL string, mntURL string) {
  if err := os.Mkdir(mntURL, 0777); err != nil {
    logrus.Errorf("Mkdir dir %s error %v", mntURL, err)
  }
  dirs := "dirs=" + rootURL + "writeLayer:" + rootURL + "busybox"
  cmd := exec.Command("mount", "-t", "aufs", "-o", dirs, "none", mntURL)
  cmd.Stdout = os.Stdout
  cmd.Stdin = os.Stdin
  if err := cmd.Run(); err != nil {
    logrus.Errorf("%v", err)
  }
}

func DeleteWorkSpace(rootURL string, mntURL string) {
  cmd := exec.Command("umount", mntURL)
  cmd.Stdout = os.Stdout
  cmd.Stdin = os.Stdin
  if err := cmd.Run(); err != nil {
    logrus.Errorf("%v", err)
  }
  if err := os.RemoveAll(mntURL); err != nil {
    logrus.Errorf("Remove Dir %s error %v", mntURL, err)
  }
}

func DeleteWriteLayer(rootURL string) {
  writeURL := rootURL + "writeLayer/"
  if err := os.RemoveAll(writeURL); err != nil {
    logrus.Errorf("Remove Dir %s error %v", writeURL, err)
  }
}
