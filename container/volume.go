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

func volumeExtract(volume string) []string {
	var volumeURLs = strings.Split(volume, ":")
	return volumeURLs
}

func NewWorkSpace(volume string, containerName string, imageName string) {

	if err := CreateReadOnlyLayer(imageName); err != nil {
		logrus.Infof("create read only layer error %v", err)
	}

	CreateWriteLayer(containerName)
	if err := CreateMountPoint(containerName, imageName); err != nil {
		logrus.Errorf("create mount point error %v", err)
		os.Exit(1)
	}

	if volume != "" {
		volumeURLs := volumeExtract(volume)
		length := len(volumeURLs)
		if length == 2 && volumeURLs[0] != "" && volumeURLs[1] != "" {
			if err := MountVolume(volumeURLs, containerName); err != nil {
				logrus.Errorf("mount volume error %v", err)
				os.Exit(1)
			}
			logrus.Infof("volume mounted: %q", volumeURLs)
		} else {
			logrus.Infof("volume parameter input is not correct.")
		}
	}
}

func MountVolume(volumeURLs []string, containerName string) error {
	// 创建宿主机文件目录
	parentUrl := volumeURLs[0]
	if err := os.MkdirAll(parentUrl, 0777); err != nil {
		logrus.Infof("MkdirAll parent dir %s error. %v", parentUrl, err)
		return err
	}
	// 在容器文件系统里创建挂载点
	containerUrl := volumeURLs[1]
	mntUrl := fmt.Sprintf(MntUrl, containerName)
	containerVolumeURL := mntUrl + "/" + containerUrl
	if err := os.MkdirAll(containerVolumeURL, 0777); err != nil {
		logrus.Infof("MkdirAll container dir %s error. %v", containerVolumeURL, err)
		return err
	}
	// 把宿主机文件目录挂载到容器挂载点
	dirs := "dirs=" + parentUrl
	_, err := exec.Command("mount", "-t", "aufs", "-o", dirs, "none", containerVolumeURL).CombinedOutput()
	if err != nil {
		logrus.Errorf("mount volume failed. %v", err)
		return err
	}
	return nil
}

// 将busybox.tar解压到busybox目录,作为容器的只读层
func CreateReadOnlyLayer(imageName string) error {
	untarFolderUrl := RootUrl + "/images/" + imageName + "/"
	imageUrl := RootUrl + "/images/" + imageName + ".tar"
	exist := utils.PathExists(untarFolderUrl)
	if !exist {
		if err := os.MkdirAll(untarFolderUrl, 0777); err != nil {
			return fmt.Errorf("mkdir dir %s error. %v", untarFolderUrl, err)
		}
		if _, err := exec.Command("tar", "-xvf", imageUrl, "-C", untarFolderUrl).CombinedOutput(); err != nil {
			return fmt.Errorf("untar dir %s, error %v", imageUrl, err)
		}
	} else {
		return fmt.Errorf("path %s exist", untarFolderUrl)
	}
	return nil
}

// 创建可写层
func CreateWriteLayer(containerName string) {
	writeUrl := fmt.Sprintf(WriteLayerUrl, containerName)
	if utils.PathExists(writeUrl) {
		os.RemoveAll(writeUrl)
	}
	if err := os.MkdirAll(writeUrl, 0777); err != nil {
		logrus.Errorf("Mkdir dir %s error %v", writeUrl, err)
	}
}

// 创建挂载点
func CreateMountPoint(containerName string, imageName string) error {
	mntUrl := fmt.Sprintf(MntUrl, containerName)
	if utils.PathExists(mntUrl) {
		res, _ := exec.Command("sh", "-c", "mount | grep", mntUrl).Output()
		if string(res) != "" {
			if err := syscall.Unmount(mntUrl, 0); err != nil {
				logrus.Errorf("umount %s error: %v", mntUrl, err)
				DeleteWriteLayer(containerName)
				return err
			}
		}
		os.RemoveAll(mntUrl)
	}
	if err := os.MkdirAll(mntUrl, 0777); err != nil {
		logrus.Errorf("Mkdir dir %s error %v", mntUrl, err)
		return err
	}
	writeUrl := fmt.Sprintf(WriteLayerUrl, containerName)
	imageLocation := RootUrl + "/images/" + imageName
	dirs := "dirs=" + writeUrl + ":" + imageLocation
	cmd := exec.Command("mount", "-t", "aufs", "-o", dirs, "none", mntUrl)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		logrus.Errorf("mount %s error %v", mntUrl, err)
		return err
	}
	return nil
}

func DeleteWorkSpace(volume, containerName string) {
	if volume != "" {
		volumeURLs := volumeExtract(volume)
		length := len(volumeURLs)
		if length == 2 && volumeURLs[0] != "" && volumeURLs[1] != "" {
			if err := DeleteMountPointWithVolume(volumeURLs, containerName); err != nil {
				logrus.Errorf("DeleteMountPointWithVolume error %v", err)
				os.Exit(1)
			}
		} else {
			if err := DeleteMountPoint(containerName); err != nil {
				logrus.Errorf("DeleteMountPoint error %v", err)
				os.Exit(1)
			}
		}
	} else {
		if err := DeleteMountPoint(containerName); err != nil {
			logrus.Errorf("DeleteMountPoint error %v", err)
			os.Exit(1)
		}
	}
	DeleteWriteLayer(containerName)
}

func DeleteMountPointWithVolume(volumeURLs []string, containerName string) error {
	// 卸载容器里volume挂载点的文件系统
	mntUrl := fmt.Sprintf(MntUrl, containerName)
	containerUrl := mntUrl + "/" + volumeURLs[1]
	if err := syscall.Unmount(containerUrl, 0); err != nil {
		logrus.Errorf("umount %s error: %v", containerUrl, err)
		return err
	}
	return DeleteMountPoint(containerName)
}

func DeleteMountPoint(containerName string) error {
	mntUrl := fmt.Sprintf(MntUrl, containerName)
	if err := syscall.Unmount(mntUrl, 0); err != nil {
		logrus.Errorf("umount %s error %v", containerName, err)
		return err
	}
	// Even though we just unmounted the filesystem, AUFS will prevent deleting the mntpoint
	// for some time. We'll just keep retrying until it succeeds.
	for retries := 0; retries < 1000; retries++ {
		err := os.RemoveAll(mntUrl)
		if err == nil {
			return nil
		}
		if os.IsNotExist(err) {
			return err
		}
		time.Sleep(10 * time.Millisecond)
	}
	logrus.Errorf("failed to umount %s", mntUrl)
	return fmt.Errorf("failed to umount %s", mntUrl)
}

func DeleteWriteLayer(containerName string) {
	writeUrl := fmt.Sprintf(WriteLayerUrl, containerName)
	if err := os.RemoveAll(writeUrl); err != nil {
		logrus.Errorf("Remove Dir %s error %v", writeUrl, err)
	}
}
