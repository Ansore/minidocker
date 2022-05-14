package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"minidocker/cgroups"
	"minidocker/cgroups/subsystems"
	"minidocker/container"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

func sendInitCommand(cmdArr []string, writePipe *os.File) {
	command := strings.Join(cmdArr, " ")
	logrus.Infof("command all is %s", command)
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

func recordContainerInfo(containerPid int, commandArray []string, containerName string) (string, error) {
	// 生成10位数字的容器ID
	id := randStringBytes(10)
	// 以当前时间为容器创建时间
	createTime := time.Now().Format("2006-01-01 14:00:00")
	command := strings.Join(commandArray, "")
	if containerName == "" {
		containerName = id
	}
	// 生成容器信息结构体实例
	containerInfo := &container.ContainerInfo{
		Id:         id,
		Pid:        strconv.Itoa(containerPid),
		Command:    command,
		CreateTime: createTime,
		Status:     container.RUNNING,
		Name:       containerName,
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

func Run(tty bool, cmdArr []string, resConf *subsystems.ResourceConfig, volume string, containerName string) {
	childProcess, writePipe := container.NewParentProcess(tty, containerName, volume)
	if childProcess == nil {
		logrus.Errorf("New parent process error")
		return
	}
	if err := childProcess.Start(); err != nil {
		logrus.Error(err)
	}
	containerName, err := recordContainerInfo(childProcess.Process.Pid, cmdArr, containerName)
	if err != nil {
		logrus.Errorf("Record container info error %v", err)
		return
	}

	cgroupManager := cgroups.NewCgroupManager("minidocker-cgroup")
	defer cgroupManager.Destroy()
	if err := cgroupManager.Set(resConf); err != nil {
		logrus.Errorf("cgroupManager set resConf error %v", err)
	}
	if err := cgroupManager.Apply(childProcess.Process.Pid); err != nil {
		logrus.Errorf("cgroupManager Apply childProcess %d error %v", childProcess.Process.Pid, err)
	}
	sendInitCommand(cmdArr, writePipe)
	if tty {
		if err := childProcess.Wait(); err != nil {
			logrus.Errorf("parent Wait error %v", err)
		}
		mntURL := "/root/mnt"
		rootURL := "/root/"
		container.DeleteWorkSpace(rootURL, mntURL, volume)
		deleteContainerInfo(containerName)
	}
	os.Exit(0)
}
