package network

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

type BridgeNetworkDriver struct {
}

func (d *BridgeNetworkDriver) Name() string {
	return "bridge"
}

// create linux bridge device
func createBridgeInterface(bridgeName string) error {
  // 检查是否存在同名设备
	_, err := net.InterfaceByName(bridgeName)
  // 如果已存在,返回错误
	if err == nil || !strings.Contains(err.Error(), "no such network interface") {
		return err
	}

	// create *netlink.Bridge object
  // 初始化一个netlink的link基础对象,link的名字即bridge虚拟设备的名字
	la := netlink.NewLinkAttrs()
	la.Name = bridgeName

  // 使用刚才创建的link的属性创建netlink的bridge对象
	br := &netlink.Bridge{LinkAttrs: la}
  // 调用netlink的linkadd方法,创建bridge虚拟网络设备
  // 相当于 ip link add xxx
	if err := netlink.LinkAdd(br); err != nil {
		return fmt.Errorf("bridge creation failed for bridge %s: %v", bridgeName, err)
	}
	return nil
}

// 设置网络接口位UP状态
func setInterfaceUP(interfaceName string) error {
	iface, err := netlink.LinkByName(interfaceName)
	if err != nil {
		return fmt.Errorf("error retrieving a link named [ %s ]: %v", iface.Attrs().Name, err)
	}

  // 通过netlink的LinkSetUp接口将状态设置为up
  // 相当于 ip link set xxx up
	if err := netlink.LinkSetUp(iface); err != nil {
		return fmt.Errorf("error enabling interface for %s: %v", interfaceName, err)
	}
	return nil
}

// set the ip addr of a netlink interface
func setInterfaceIP(name string, rawIP string) error {
	retries := 2
	var iface netlink.Link
	var err error

	for i := 0; i < retries; i++ {
    // 通过netlink的LinkByName方法找到需要设置的网络接口
		iface, err = netlink.LinkByName(name)
		if err == nil {
			break
		}
		logrus.Debugf("error retrieving new bridge netlink link [ %s ]... retrying", name)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return fmt.Errorf("abandoning retrieving the new bridge link from netlink, Run [ ip link ] to troubleshoot the error: %v", err)
	}
  // ipnet包含网段的信息和原始IP
	ipNet, err := netlink.ParseIPNet(rawIP)
	if err != nil {
		return err
	}
	addr := &netlink.Addr{
		IPNet:       ipNet,
		Label:       "",
		Flags:       0,
		Scope:       0,
	}
  // AddrAdd相当于 ip addr add xxx
  // 如果配置了地址所在的网段的信息,如192.168.0.0/24
  // 还会配置路由表192.168.0.0/24转发到这个testbridge的网络接口上
	return netlink.AddrAdd(iface, addr)
}

// 设置iptables对应的bridge的MASQUERADE规则
func setupIPTables(bridgeName string, subnet *net.IPNet) error {
  // 创建iptables命令
  // iptables -t nat -A POSTROUTING -s <bridgeName> ! -o <bridgeName> -j MASQUERADE
	iptablesCmd := fmt.Sprintf("-t nat -A POSTROUTING -s %s ! -o %s -j MASQUERADE", subnet.String(), bridgeName)
	cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
  // 执行iptables命令配置SNAT规则
	output, err := cmd.Output()
	if err != nil {
		logrus.Errorf("iptables output, %v", output)
	}
	return err
}

// 初始化Bridge
// 初始化bridge虚拟设备->设置bridge设备的地址和路由->启动bridge->设置iptables SNAT规则
func (d *BridgeNetworkDriver) initBridge(n *Network) error {
	// try to get bridge by name, if it already exists then just exit
	bridgeName := n.Name
	if err := createBridgeInterface(bridgeName); err != nil {
		return fmt.Errorf("error add bridge: %s, error: %v", bridgeName, err)
	}

	// set bridge IP and router
	gatewayIP := *n.IpRange
	gatewayIP.IP = n.IpRange.IP

	if err := setInterfaceIP(bridgeName, gatewayIP.String()); err != nil {
		return fmt.Errorf("error assigning address: %s on bridge: %s with an eror of: %v", gatewayIP, bridgeName, err)
	}

  // start bridge dev
	if err := setInterfaceUP(bridgeName); err != nil {
		return fmt.Errorf("error set bridge up: %s, error: %v", bridgeName, err)
	}

	// setup tables
	if err := setupIPTables(bridgeName, n.IpRange); err != nil {
		return fmt.Errorf("error setting iptables for %s: %v", bridgeName, err)
	}
	return nil
}

func (d *BridgeNetworkDriver) Create(subnet string, name string) (*Network, error) {
  // 获取网段字符串中的网关IP地址和网络的IP段
	ip, ipRange, _ := net.ParseCIDR(subnet)
	ipRange.IP = ip
  // 初始化网络对象
	n := &Network{
		Name:    name,
		IpRange: ipRange,
		Driver:  d.Name(),
	}
	err := d.initBridge(n)
	if err != nil {
		logrus.Errorf("error init bridge: %v", err)
	}
  // 返回配置好的网络
	return n, err
}

func (d *BridgeNetworkDriver) Delete(network Network) error {
  // 网络名即linux bridge设备名
	bridgeName := network.Name
  // 通过netlink的LinkByName获取对应的设备
	br, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return err
	}
  // 删除linux bridge设备
	return netlink.LinkDel(br)
}

func (d *BridgeNetworkDriver) Disconnect(network Network, endpoint *Endpoint) error {
	return nil
}

func (d *BridgeNetworkDriver) Connect(network *Network, endpoint *Endpoint) error {
  // 获取linux bridge的接口名
	bridgeName := network.Name
  // 通过接口名获取到Linux bridge接口的对象和属性
	br, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return err
	}

  // 创建veth属性
	la := netlink.NewLinkAttrs()
  // 由于Linux接口名限制，名字取endpoint id的前5位
	la.Name = endpoint.ID[:5]
  // 通过设置veth接口的master属性,设置这个veth的一端挂载到网络对应的linux bridge上
	la.MasterIndex = br.Attrs().Index

  // 创建veth对象,通过PeerName配置veth另一端的接口名
  // 配置veth另一端的名字cif-{endpoint ID的前5位}
	endpoint.Device = netlink.Veth{
		LinkAttrs: la,
		PeerName:  "cif-" + endpoint.ID[:5],
	}

  // 调用netlink的linkadd方法创建这个veth接口
  // 上面指定了link的MasterIndex是网络对应的linux bridge
  // 所以veth另一端就已经挂载到了网络对应的linux bridge上
	if err = netlink.LinkAdd(&endpoint.Device); err != nil {
		return fmt.Errorf("error Add Endpoint Device: %v", err)
	}

  // 调用netlink的LinkSetUp方法,设置veth启动
  // 相当于ip link set xxx up命令
	if err = netlink.LinkSetUp(&endpoint.Device); err != nil {
		return fmt.Errorf("error Add Endpoint Device: %v", err)
	}
	return nil
}
