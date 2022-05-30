package network

import (
	"encoding/json"
	"fmt"
	"minidocker/container"
	"net"
	"os"
	"path"

	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

// 网络端点
type Endpoint struct {
	ID          string           `json:"id"`
	Device      netlink.Veth     `json:"dev"`
	IPAddress   net.IP           `json:"ip"`
	MacAddress  net.HardwareAddr `json:"mac"`
	Network     *Network
	PortMapping []string
}

var (
  defaultEndpointPath = "/var/run/minidocker/network/endpoints/"
	// 各个网络驱动的实例字典
  endpoints = map[string]*Endpoint{}
)

func GetEndpointId(networkName string, cinfo *container.ContainerInfo) string {
  return fmt.Sprintf("%s-%s", cinfo.Id, networkName)
}

func (ep *Endpoint) dump(dumpPath string) error {
	// 检查保存目录是否存在,不存在则创建
	if _, err := os.Stat(dumpPath); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(dumpPath, 0644); err != nil {
				logrus.Errorf("makedir %s, error %v", dumpPath, err)
			}
		} else {
			return err
		}
	}

	// 保存的文件名是endpoint id
	epPath := path.Join(dumpPath, ep.ID)
	// 打开保存的文件用于写入,存在则内容清空,只写入,不存在则创建
	nwFile, err := os.OpenFile(epPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		logrus.Errorf("error: %v", err)
		return err
	}
	defer nwFile.Close()

	// 通过json的库序列化网络对象到json的字符串
	epJson, err := json.Marshal(ep)
	if err != nil {
		logrus.Errorf("error: %v", err)
		return err
	}

	// 写入文件
	_, err = nwFile.Write(epJson)
	if err != nil {
		logrus.Errorf("error: %v", err)
		return err
	}
	return nil
}

// 删除网络配置文件
func (ep *Endpoint) remove(dumpPath string) error {
	if _, err := os.Stat(path.Join(dumpPath, ep.ID)); err != nil {
		if os.IsNotExist(err) {
			return nil
		} else {
			return err
		}
	} else {
		return os.Remove(path.Join(dumpPath, ep.ID))
	}
}

func (ep *Endpoint) load(dumpPath string) error {
	// 打开配置文件
	nwConfigFile, err := os.Open(dumpPath)
	if err != nil {
		return err
	}
	defer nwConfigFile.Close()

	// 从配置文件中读取网络的配置json字符串
	epJson := make([]byte, 2000)
	n, err := nwConfigFile.Read(epJson)
	if err != nil {
		return err
	}

	// json字符串反序列换出网络配置
	err = json.Unmarshal(epJson[:n], ep)
	if err != nil {
		logrus.Errorf("Error load nw info %v", err)
		return err
	}
	return nil
}
