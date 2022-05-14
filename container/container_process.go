package container

import (
	"fmt"
	"minidocker/utils"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
)

type ContainerInfo struct {
	Pid        string `json:"pid"`
	Id         string `json:"id"`
	Name       string `json:"name"`
	Command    string `json:"command"`
	CreateTime string `json:"createTime"`
	Status     string `json:"status"`
}

var (
	RUNNING             string = "running"
	STOP                string = "stopped"
	Exit                string = "exited"
	DefaultInfoLocation string = "/var/run/minidocker/%s/"
	ConfigName          string = "config.json"
	ContainerLogFile    string = "container.log"
)

func NewPipe() (*os.File, *os.File, error) {
	read, write, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}
	return read, write, err
}

func NewParentProcess(tty bool, containerName string, volume string) (*exec.Cmd, *os.File) {
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
	} else {
		// 生成容器对应目录的container.log文件
		dirURL := fmt.Sprintf(DefaultInfoLocation, containerName)
		if err := os.MkdirAll(dirURL, 0622); err != nil {
			logrus.Errorf("NewParentProcess mkdir %s error %v", dirURL, err)
			return nil, nil
		}
		stdLogFilePath := dirURL + ContainerLogFile
		stdLogFile, err := os.Create(stdLogFilePath)
		if err != nil {
			logrus.Errorf("NewParentProcess create file %s error %v", stdLogFilePath, err)
			return nil, nil
		}
		// 生成好的文件赋值给stdout, 这样就能把容器里的标准输出重定向到这个文件中
		cmd.Stdout = stdLogFile
	}

	cmd.ExtraFiles = []*os.File{readPipe}
	mntURL := "/root/mnt"
	rootURL := "/root/"
	NewWorkSpace(rootURL, mntURL, volume)
	cmd.Dir = mntURL
	return cmd, writePipe
}

func volumeExtract(volume string) []string {
	var volumeURLs []string
	volumeURLs = strings.Split(volume, ":")
	return volumeURLs
}

func NewWorkSpace(rootURL string, mntURL string, volume string) {
	busyboxURL := rootURL + "busybox/"
	busyboxTarURL := rootURL + "busybox.tar"
	writeLayerURL := rootURL + "writeLayer/"

	CreateReadOnlyLayer(busyboxURL, busyboxTarURL)
	CreateWriteLayer(writeLayerURL)
	CreateMountPoint(writeLayerURL, busyboxURL, mntURL)

	if volume != "" {
		volumeURLs := volumeExtract(volume)
		length := len(volumeURLs)
		if length == 2 && volumeURLs[0] != "" && volumeURLs[1] != "" {
			MountVolume(rootURL, mntURL, volumeURLs)
			logrus.Infof("volume mounted: %q", volumeURLs)
		} else {
			logrus.Infof("volume parameter input is not correct.")
		}
	}
}

func MountVolume(rootURL string, mntURL string, volumeURLs []string) {
	// 创建宿主机文件目录
	parentUrl := volumeURLs[0]
	if err := os.Mkdir(parentUrl, 0777); err != nil {
		logrus.Infof("Mkdir parent dir %s error. %v", parentUrl, err)
	}
	// 在容器文件系统里创建挂载点
	containerUrl := volumeURLs[1]
	containerVolumeURL := mntURL + containerUrl
	if err := os.Mkdir(containerVolumeURL, 0777); err != nil {
		logrus.Infof("Mkdir container dir %s error. %v", containerUrl, err)
	}
	// 把宿主机文件目录挂载到容器挂载点
	dirs := "dirs=" + parentUrl
	cmd := exec.Command("mount", "-t", "aufs", "-o", dirs, "none", containerVolumeURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logrus.Errorf("mount volume failed. %v", err)
	}
}

// 将busybox.tar解压到busybox目录,作为容器的只读层
func CreateReadOnlyLayer(busyboxURL string, busyboxTarURL string) {
	exist := utils.PathExists(busyboxURL)
	if exist {
		if err := os.Mkdir(busyboxURL, 0777); err != nil {
			logrus.Errorf("Mkdir dir %s error. %v", busyboxURL, err)
      os.Exit(1)
		}
		if _, err := exec.Command("tar", "-xvf", busyboxTarURL, "-C", busyboxURL).CombinedOutput(); err != nil {
			logrus.Errorf("Untar dir %s, error %v", busyboxTarURL, err)
      os.Exit(1)
		}
	}
}

// 创建可写层
func CreateWriteLayer(writeLayerURL string) {
	if utils.PathExists(writeLayerURL) {
		os.RemoveAll(writeLayerURL)
	}
	if err := os.Mkdir(writeLayerURL, 0777); err != nil {
		logrus.Errorf("Mkdir dir %s error %v", writeLayerURL, err)
	}
}

// 创建挂载点
func CreateMountPoint(writeLayerURL string, busyboxURL string, mntURL string) {
	if utils.PathExists(mntURL) {
    res, _ := exec.Command("sh", "-c", "mount | grep", mntURL).Output()
		if string(res) != "" {
			if err := syscall.Unmount(mntURL, 0); err != nil {
				logrus.Errorf("umount %s error: %v", mntURL, err)
        DeleteWriteLayer(writeLayerURL)
        os.Exit(1)
			}
		}
		os.RemoveAll(mntURL)
	}
	if err := os.Mkdir(mntURL, 0777); err != nil {
		logrus.Errorf("Mkdir dir %s error %v", mntURL, err)
	}
	dirs := "dirs=" + writeLayerURL + ":" + busyboxURL
	cmd := exec.Command("mount", "-t", "aufs", "-o", dirs, "none", mntURL)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		logrus.Errorf("mount %s error %v", mntURL, err)
	}
}

func DeleteWorkSpace(rootURL string, mntURL string, volume string) {
	if volume != "" {
		volumeURLs := volumeExtract(volume)
		length := len(volumeURLs)
		if length == 2 && volumeURLs[0] != "" && volumeURLs[1] != "" {
			DeleteMountPointWithVolume(mntURL, volumeURLs)
		} else {
			DeleteMountPoint(mntURL)
		}
	} else {
		DeleteMountPoint(mntURL)
	}
	DeleteWriteLayer(rootURL)
}

func DeleteMountPointWithVolume(mntURL string, volumeURLs []string) {
	// 卸载容器里volume挂载点的文件系统
	containerUrl := mntURL + volumeURLs[1]
	if err := syscall.Unmount(containerUrl, 0); err != nil {
		logrus.Errorf("umount %s error: %v", containerUrl, err)
	}
	DeleteMountPoint(mntURL)
}

func DeleteMountPoint(mntURL string) {
	if err := syscall.Unmount(mntURL, 0); err != nil {
		logrus.Errorf("umount %s error %v", mntURL, err)
	}
	// Even though we just unmounted the filesystem, AUFS will prevent deleting the mntpoint
	// for some time. We'll just keep retrying until it succeeds.
	for retries := 0; retries < 1000; retries++ {
		err := os.RemoveAll(mntURL)
		if err == nil {
			return
		}
		if os.IsNotExist(err) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	logrus.Errorf("failed to umount %s", mntURL)
}

func DeleteWriteLayer(rootURL string) {
	writeURL := rootURL + "writeLayer/"
	if err := os.RemoveAll(writeURL); err != nil {
		logrus.Errorf("Remove Dir %s error %v", writeURL, err)
	}
}
