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

	if err := cmd.Run(); err != nil {
		logrus.Errorf("Exec container %s error %v", containerName, err)
	}
}
