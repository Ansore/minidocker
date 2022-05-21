package network

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"minidocker/container"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"text/tabwriter"

	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

var (
	defaultNetworkPath = "/var/run/minidocker/network/network/"
	drivers            = map[string]NetworkDriver{}
	networks           = map[string]*Network{}
)

// 网络驱动
type NetworkDriver interface {
	// 驱动名
	Name() string
	// 创建网络
	Create(subnet string, name string) (*Network, error)
	// 删除网络
	Delete(network Network) error
	// 连接容器网络端点到网络
	Connect(network *Network, endpoint *Endpoint) error
	// 从网络上移除网络端点
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
			if err := os.MkdirAll(dumpPath, 0644); err != nil {
				logrus.Errorf("makedir %s, error %v", dumpPath, err)
			}
		} else {
			return err
		}
	}

	nwPath := path.Join(dumpPath, nw.Name)
	nwFile, err := os.OpenFile(nwPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		logrus.Errorf("error: %v", err)
		return err
	}
	defer nwFile.Close()

	nwJson, err := json.Marshal(nw)
	if err != nil {
		logrus.Errorf("error: %v", err)
		return err
	}

	_, err = nwFile.Write(nwJson)
	if err != nil {
		logrus.Errorf("error: %v", err)
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
	var bridgeDriver = BridgeNetworkDriver{}
	drivers[bridgeDriver.Name()] = &bridgeDriver

	if _, err := os.Stat(defaultNetworkPath); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(defaultNetworkPath, 0644); err != nil {
				logrus.Errorf("mkdir %s, error %v", defaultNetworkPath, err)
			}
		} else {
			return err
		}
	}
  if err := filepath.Walk(defaultNetworkPath, func(nwPath string, _ fs.FileInfo, _ error) error {
		if strings.HasSuffix(nwPath, "/") {
			return nil
		}
		_, nwName := path.Split(nwPath)
		nw := &Network{
			Name: nwName,
		}
		if err := nw.load(nwPath); err != nil {
			logrus.Errorf("error load network: %s", err)
		}

		networks[nwName] = nw
		return nil
	}); err != nil {
    return err
  }
	return nil
}

func CreateNetwork(dirver, subnet, name string) error {
	// ParseCIDR是golang net包的函数，功能是将网段的字符串转换为net.IPNet的对象
	_, cidr, _ := net.ParseCIDR(subnet)
	// 通过IPAM分配网管IP,获取到网段中第一个IP作为网关的IP
	gatewayIp, err := ipAllocator.Allocate(cidr)
	if err != nil {
		return err
	}
	cidr.IP = gatewayIp

	nw, err := drivers[dirver].Create(cidr.String(), name)
	if err != nil {
		return err
	}
	return nw.dump(defaultNetworkPath)
}

func ListNetwork() {
	w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
	fmt.Fprint(w, "NAME\tIpRange\tDriver\n")
	for _, nw := range networks {
		fmt.Fprintf(w, "%s\t%s\t%s\n", nw.Name, nw.IpRange.String(), nw.Driver)
	}
	if err := w.Flush(); err != nil {
		logrus.Errorf("flush error %v", err)
		return
	}
}

func DeleteNetwork(networkName string) error {
	nw, ok := networks[networkName]
	if !ok {
		return fmt.Errorf("no such network: %s", networkName)
	}
	if err := ipAllocator.Release(nw.IpRange, &nw.IpRange.IP); err != nil {
		return fmt.Errorf("error remove network gateway ip: %s", err)
	}
	if err := drivers[nw.Driver].Delete(*nw); err != nil {
		return fmt.Errorf("error remove network driver error %v", err)
	}
	return nw.remove(defaultNetworkPath)
}

func enterContainerNetns(enLink *netlink.Link, cinfo *container.ContainerInfo) func() {
	f, err := os.OpenFile(fmt.Sprintf("/proc/%s/ns/net", cinfo.Pid), os.O_RDONLY, 0)
	if err != nil {
		logrus.Errorf("error get container net namespace, %v", err)
	}

	nsFD := f.Fd()
	runtime.LockOSThread()

	// 修改veth peer 另一端移到容器的namespace中
	if err = netlink.LinkSetNsFd(*enLink, int(nsFD)); err != nil {
		logrus.Errorf("error set link netns, error %v", err)
	}

	// 或得当前容器的namespace
	origin, err := netns.Get()
	if err != nil {
		logrus.Errorf("error get current netns, %v", err)
	}

	// 设置当前进程到新的网络namespace, 并在函数执行完成之后再恢复到之前的namespace
	if err = netns.Set(netns.NsHandle(nsFD)); err != nil {
		logrus.Errorf("error set netns, %v", err)
	}

	return func() {
    if err := netns.Set(origin); err != nil {
      logrus.Errorf("netns set error %v", err)
    }
		origin.Close()
		runtime.UnlockOSThread()
		f.Close()
	}
}

func configEndpointIpAddressAndRoute(ep *Endpoint, cinfo *container.ContainerInfo) error {
	peerLink, err := netlink.LinkByName(ep.Device.PeerName)
	if err != nil {
		return fmt.Errorf("fail config endpoint: %v", err)
	}

	defer enterContainerNetns(&peerLink, cinfo)()
	interfaceIp := *ep.Network.IpRange
	interfaceIp.IP = ep.IPAddress
	if err = setInterfaceIP(ep.Device.PeerName, interfaceIp.String()); err != nil {
		return fmt.Errorf("%s,%v", ep.Network, err)
	}
	if err = setInterfaceUP(ep.Device.PeerName); err != nil {
		return err
	}
	if err = setInterfaceUP("lo"); err != nil {
		return err
	}
	_, cidr, _ := net.ParseCIDR("0.0.0.0/0")

	defaultRoute := &netlink.Route{
		LinkIndex: peerLink.Attrs().Index,
		Gw:        ep.Network.IpRange.IP,
		Dst:       cidr,
	}
	if err = netlink.RouteAdd(defaultRoute); err != nil {
		return err
	}
	return nil
}

func configPortMapping(ep *Endpoint, _ *container.ContainerInfo) error {
	for _, pm := range ep.PortMapping {
		PortMapping := strings.Split(pm, ":")
		if len(PortMapping) != 2 {
			logrus.Errorf("port mapping format error, %v", pm)
			continue
		}
		iptablesCmd := fmt.Sprintf("-t nat -A PREROUTING -p tcp -m tcp --dport %s -j DNAT --to-destination %s:%s",
      PortMapping[0], ep.IPAddress.String(), PortMapping[1])
    cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
    output, err := cmd.Output()
    if err != nil {
      logrus.Errorf("iptables output, %v", output)
      continue
    }
	}
  return nil
}

func Connect(networkName string, cinfo *container.ContainerInfo) error {
  network, ok := networks[networkName]
  if !ok {
    return fmt.Errorf("no such network: %s", networkName)
  }

  // 分配容器IP
  ip, err := ipAllocator.Allocate(network.IpRange)
  if err != nil {
    return err
  }
  // 创建网络端点
  ep := &Endpoint {
    ID: fmt.Sprintf("%s-%s", cinfo.Id, networkName),
    IPAddress: ip,
    Network: network,
    PortMapping: cinfo.PortMapping,
  }
  // 调用网络驱动挂载和配置网络端点
  if err = drivers[network.Driver].Connect(network, ep); err != nil {
    return err
  }
  // 到容器的namespace配置容器的网络设备IP地址
  if err = configEndpointIpAddressAndRoute(ep, cinfo); err != nil {
    return nil
  }
  return configPortMapping(ep, cinfo)
}

func Disconnect(networkName string, cinfo *container.ContainerInfo) error {
  return nil
}
