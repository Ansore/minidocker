package command

import (
	"fmt"
	"minidocker/container"
	"os"

	"github.com/sirupsen/logrus"
)

func removeContainer(containerName string) {
  containerInfo, err := getContainerInfoByName(containerName)
  if err != nil {
    logrus.Errorf("Get container %s info error %v", containerName, err)
    return
  }
  if containerInfo.Status != container.STOP {
    logrus.Errorf("Cann't remove running container")
    return
  }
  dirURL := fmt.Sprintf(container.DefaultInfoLocation, containerName)
  if err := os.RemoveAll(dirURL); err != nil {
    logrus.Errorf("Remove file %s error %v", dirURL, err)
    return
  }
}
