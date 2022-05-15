package command

import (
	"fmt"
	"minidocker/container"
	"os/exec"

	"github.com/sirupsen/logrus"
)

func commitContainer(containerName,imageName string) {
  mntUrl := fmt.Sprintf(container.MntUrl, containerName)
  mntUrl += "/"
  imageTar := container.RootUrl + "/images/" + imageName + ".tar"
  if _, err := exec.Command("tar", "-czf", imageTar, "-C", mntUrl, ".").CombinedOutput(); err != nil {
    logrus.Errorf("Tar folder %s error %v", mntUrl, err)
  }
}
