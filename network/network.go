package network

import (
	"encoding/json"
	"net"
	"os"
	"path"

	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

var (
	defaultNetwordPath = "/var/run/minidocker/network/network/"
  drivers = map[string]NetworkDriver{}
  networks = map[string]*Network{}
)

type NetworkDriver interface {
  Name() string
  Create(subnet string, name string) (*Network, error)
  Delete(network Network) error
  Connect(network *Network, endpoint *Endpoint) error
  Disconnect(network Network, endpoint *Endpoint) error
}

type Network struct {
	Name    string     // 网络名
	IpRange *net.IPNet // 地址段
	Driver  string     // 网络驱动名
}

type Endpoint struct {
	ID          string           `json:"id"`
	Device      netlink.Veth     `json:"dev"`
	IPAddress   net.IP           `json:"ip"`
	MacAddress  net.HardwareAddr `json:"mac"`
	Network     *Network
	PortMapping []string
}

func (nw *Network) dump(dumpPath string) error {
  if _, err := os.Stat(dumpPath); err != nil {
    if os.IsNotExist(err) {
      os.MkdirAll(dumpPath, 0644)
    } else {
      return err
    }
  }

  nwPath := path.Join(dumpPath, nw.Name)
  nwFile, err := os.OpenFile(nwPath, os.O_TRUNC | os.O_WRONLY | os.O_CREATE, 0644)
  if err != nil {
    logrus.Errorf("error: %v", err)
    return err
  }
  defer nwFile.Close()

  nwJson, err := json.Marshal(nw)
  if err != nil {
    logrus.Errorf("error: ", err)
    return err
  }

  _, err = nwFile.Write(nwJson)
  if err != nil {
    logrus.Errorf("error: ", err)
    return err
  }
  return nil
}

func (nw *Network) remove(dumpPath string) error {
  if _, err := os.Stat(path.Join(dumpPath, nw.Name)); err != nil {
    if os.IsNotExist(err) {
      return nil
    } else {
      return err
    }
  } else {
    return os.Remove(path.Join(dumpPath, nw.Name))
  }
}

func (nw *Network) load(dumpPath string) error {
  nwConfigFile, err := os.Open(dumpPath)
  if err != nil {
    return err
  }
  defer nwConfigFile.Close()

  nwJson := make([]byte, 2000)
  n, err := nwConfigFile.Read(nwJson)
  if err != nil {
    return err
  }

  err = json.Unmarshal(nwJson[:n], err)
  if err != nil {
    logrus.Errorf("Error load nw info %v", err)
    return err
  }
  return nil
}

func Init() error {
  var bridgeDriver = 
}
