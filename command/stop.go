package command

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"minidocker/container"
	"strconv"
	"syscall"

	"github.com/sirupsen/logrus"
)

func getContainerInfoByName(containerName string) (*container.ContainerInfo, error) {
  // 构造存放容器信息的路径
  dirURL := fmt.Sprintf(container.DefaultInfoLocation, containerName)
  configFilePath := dirURL + container.ConfigName
  contentBytes, err := ioutil.ReadFile(configFilePath)
  if err != nil {
    logrus.Errorf("Read file %s error %v", configFilePath, err)
    return nil, err
  }
  var containerInfo container.ContainerInfo
  // 将容器信息字符串反序列化
  if err := json.Unmarshal(contentBytes, &containerInfo); err != nil {
    logrus.Errorf("GetContainerInfoByName unmarshal error %v", err)
    return nil, err
  }
  return &containerInfo, nil
}

func stopContainer(containerName string) {
  // 根据容器名获取对应的进程ID
  pid, err := getContainerPidByName(containerName)
  if err != nil {
    logrus.Errorf("Get container pid by name %s error %v", containerName, err)
    return
  }
  // 将string的pid转换为int
  pidInt, err := strconv.Atoi(pid)
  if err != nil {
    logrus.Errorf("Conver pid from string to int error %v", err)
    return
  }
  // 调用kill发送信号给进程,结束进程
  if err := syscall.Kill(pidInt, syscall.SIGTERM); err != nil {
    logrus.Errorf("Stop container %s error %v", containerName, err)
    // return
  }
  // 根据容器名获取对应的信息对象
  containerInfo, err := getContainerInfoByName(containerName)
  if err != nil {
		logrus.Errorf("Get container %s info error %v", containerName, err)
		return
  }
  containerInfo.Status = container.STOP
  containerInfo.Pid = ""
  newContentBytes, err := json.Marshal(containerInfo)
  if err != nil {
		logrus.Errorf("Json marshal %s error %v", containerName, err)
		return
  }
  dirURL := fmt.Sprintf(container.DefaultInfoLocation, containerName)
  configFilePath := dirURL + container.ConfigName
  if err := ioutil.WriteFile(configFilePath, newContentBytes, 0622); err != nil {
    logrus.Errorf("Write file %s error %v", configFilePath, err)
    return
  }
}
