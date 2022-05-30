package command

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"minidocker/cgroups"
	"minidocker/cgroups/subsystems"
	"minidocker/container"
	"minidocker/network"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

func sendInitCommand(cmdArr []string, writePipe *os.File) {
	command := strings.Join(cmdArr, " ")
	if _, err := writePipe.WriteString(command); err != nil {
		logrus.Errorf("Write Pipe write command error %v", err)
	}
	writePipe.Close()
}

func randStringBytes(n int) string {
	letterBytes := "1234567890"
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func recordContainerInfo(containerPid int, commandArray []string, containerName string, containerId string, volume string) (string, error) {
	// 以当前时间为容器创建时间
	createTime := time.Now().Format("2006-01-01 14:00:00")
	command := strings.Join(commandArray, "")
	// 生成容器信息结构体实例
	containerInfo := &container.ContainerInfo{
		Id:         containerId,
		Pid:        strconv.Itoa(containerPid),
		Command:    command,
		CreateTime: createTime,
		Status:     container.RUNNING,
		Name:       containerName,
		Volume:     volume,
	}

	// 将容器信息序列化成字符串
	jsonBytes, err := json.Marshal(containerInfo)
	if err != nil {
		logrus.Errorf("Record ContainerInfo error %v", err)
	}
	jsonStr := string(jsonBytes)
	// 拼凑存储容器信息的路径
	dirUrl := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	// 如果路径不存在，级联全部创建
	if err := os.MkdirAll(dirUrl, 0622); err != nil {
		logrus.Errorf("Makedir %s error %v", dirUrl, err)
		return "", err
	}
	fileName := dirUrl + "/" + container.ConfigName
	// 创建最终的配置文件
	file, err := os.Create(fileName)
	if err != nil {
		logrus.Errorf("create file %s error %v", fileName, err)
	}
	defer file.Close()
	if err != nil {
		logrus.Errorf("Create file %s error %v", fileName, err)
	}
	// json序列化后的数据写入文件中
	if _, err := file.WriteString(jsonStr); err != nil {
		logrus.Errorf("File write string error %v", err)
		return "", err
	}
	return containerName, err
}

func deleteContainerInfo(containerId string) {
	dirURL := fmt.Sprintf(container.DefaultInfoLocation, containerId)
	if err := os.RemoveAll(dirURL); err != nil {
		logrus.Errorf("Remove dir %s error %v", dirURL, err)
	}
}

func Run(tty bool, cmdArr []string, resConf *subsystems.ResourceConfig, volume string, containerName string, imageName string, envSlice []string, nw string, portmapping []string) {
	containerId := randStringBytes(10)
	if containerName == "" {
		containerName = containerId
	}
	childProcess, writePipe := container.NewParentProcess(tty, containerName, volume, imageName, envSlice)
	if childProcess == nil {
		logrus.Errorf("New parent process error")
		return
	}
	if err := childProcess.Start(); err != nil {
		logrus.Error(err)
	}
	containerName, err := recordContainerInfo(childProcess.Process.Pid, cmdArr, containerName, containerId, volume)
	if err != nil {
		logrus.Errorf("Record container info error %v", err)
		return
	}

	// use containerId as cgroup name
	cgroupManager := cgroups.NewCgroupManager(containerId)
	defer cgroupManager.Destroy()
	if err := cgroupManager.Set(resConf); err != nil {
		logrus.Errorf("cgroupManager set resConf error %v", err)
	}
	if err := cgroupManager.Apply(childProcess.Process.Pid); err != nil {
		logrus.Errorf("cgroupManager Apply childProcess %d error %v", childProcess.Process.Pid, err)
	}

	containerInfo := &container.ContainerInfo{
		Id:          containerId,
		Pid:         strconv.Itoa(childProcess.Process.Pid),
		Name:        containerName,
		PortMapping: portmapping,
	}
	// network
	if nw != "" {
		// config container network
		if err := network.Init(); err != nil {
			logrus.Errorf("network init error %v", err)
		}

		if err := network.Connect(nw, containerInfo); err != nil {
			logrus.Errorf("error connect network %v", err)
			return
		}
	}

	sendInitCommand(cmdArr, writePipe)
	if tty {
		if err := childProcess.Wait(); err != nil {
			logrus.Errorf("parent Wait error %v", err)
		}
		container.DeleteWorkSpace(volume, containerName)
		deleteContainerInfo(containerName)
		network.Disconnect(nw, containerInfo)
	}
	os.Exit(0)
}
