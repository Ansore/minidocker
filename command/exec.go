package command

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"minidocker/container"
	_ "minidocker/nsenter"

	"os"
	"os/exec"
	"strings"

	"github.com/sirupsen/logrus"
)

const ENV_EXEC_PID = "minidocker_pid"
const ENV_EXEC_CMD = "minidocker_command"

func getEnvsByPid(pid string) []string {
  // 进程的环境变量获取地址 /proc/xx/environ
  path := fmt.Sprintf("/proc/%s/environ", pid)
  contentBytes, err := ioutil.ReadFile(path)
  if err != nil {
    logrus.Errorf("ReadFile %s error %v", path, err)
    return nil
  }
  envs := strings.Split(string(contentBytes), "\u0000")
  return envs
}

func getContainerPidByName(containerName string) (string, error) {
	dirURL := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	configFilePath := dirURL + container.ConfigName
	contentBytes, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return "", err
	}
	var containerInfo container.ContainerInfo
	if err := json.Unmarshal(contentBytes, &containerInfo); err != nil {
		return "", err
	}
	return containerInfo.Pid, nil
}

func ExecContainer(containerName string, comArray []string) {
	pid, err := getContainerPidByName(containerName)
	if err != nil {
		logrus.Errorf("Exec container getContainerPidByName %s error %v", containerName, err)
		return
	}
	cmdStr := strings.Join(comArray, " ")
	logrus.Infof("container pid %s", pid)
	logrus.Infof("command %s", cmdStr)

	cmd := exec.Command("/proc/self/exe", "exec")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	os.Setenv(ENV_EXEC_PID, pid)
	os.Setenv(ENV_EXEC_CMD, cmdStr)

  // 获取环境变量
  containerEnvs := getEnvsByPid(pid)
  // 设置环境变量
  cmd.Env = append(os.Environ(), containerEnvs...)

	if err := cmd.Run(); err != nil {
		logrus.Errorf("Exec container %s error %v", containerName, err)
	}
}
