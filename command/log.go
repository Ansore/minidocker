package command

import (
	"fmt"
	"io/ioutil"
	"minidocker/container"
	"os"

	"github.com/sirupsen/logrus"
)

func logContainer(containerName string) {
  // 找到文件夹对应位置
  dirURL := fmt.Sprintf(container.DefaultInfoLocation, containerName)
  logFileLocation := dirURL + container.ContainerLogFile
  // 打开日志文件
  file, err := os.Open(logFileLocation)
  if err != nil {
    logrus.Errorf("open log file %s error %v", logFileLocation, err)
  }
  defer file.Close()
  if err != nil {
    logrus.Errorf("Log container open file %s error %v", logFileLocation, err)
    return
  }
  // 将文件内的内容都读取出来
  content, err := ioutil.ReadAll(file)
  if err != nil {
    logrus.Errorf("Log container read file %s error %v", logFileLocation, err)
    return
  }
  // 输出到标准输出
  fmt.Fprint(os.Stdout, string(content) + "\n")
}
