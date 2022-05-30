package network

import (
	"encoding/json"
	"net"
	"os"
	"path"
	"strings"

	"github.com/sirupsen/logrus"
)

const ipamDefaultAllocatorPath = "/var/run/minidocker/network/ipam/subnet.json"

// 存放IP分配信息
type IPAM struct {
	// 分配文件存放位置
	SubnetAllocatorPath string
	// 网段和位图算法的数组map,key是网段,value是分配的位图数组
	Subnets *map[string]string
}

// 初始化IPAM对象
var ipAllocator = &IPAM{
	SubnetAllocatorPath: ipamDefaultAllocatorPath,
}

// 加载网络地址分配信息
func (ipam *IPAM) load() error {
	// 检查文件是否存在,如果不存在,则不需要加载
	if _, err := os.Stat(ipam.SubnetAllocatorPath); err != nil {
		if os.IsNotExist(err) {
			return err
		} else {
			return nil
		}
	}
	// 打开文件并读取
	subnetConfigFile, err := os.Open(ipam.SubnetAllocatorPath)
	if err != nil {
		return err
	}
	defer subnetConfigFile.Close()
	subnetJson := make([]byte, 2000)
	n, err := subnetConfigFile.Read(subnetJson)
	if err != nil {
		return nil
	}
	// 将文件中的内容反序列化为IP分配信息
	err = json.Unmarshal(subnetJson[:n], ipam.Subnets)
	if err != nil {
		logrus.Errorf("error dump allocation info %v", err)
		return err
	}
	return nil
}

// 存储网段地址分配信息
func (ipam *IPAM) dump() error {
	// 检查文件是否存在,如果不存在则创建
	ipamConfigFileDir, _ := path.Split(ipam.SubnetAllocatorPath)
	if _, err := os.Stat(ipamConfigFileDir); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(ipamConfigFileDir, 0644); err != nil {
				return err
			}
		} else {
			return err
		}
	}
	// 打开文件,O_TRUNC如果存在则清空,O_CREATE如果不存在则创建
	subnetConfigFile, err := os.OpenFile(ipam.SubnetAllocatorPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer subnetConfigFile.Close()

	// 将IPAM对象序列化为json
	ipamConfigJson, err := json.Marshal(ipam.Subnets)
	if err != nil {
		return nil
	}

	// 将序列化后的字符写入文件
	_, err = subnetConfigFile.Write(ipamConfigJson)
	if err != nil {
		return err
	}

	return nil
}

// 在网段中分配一个可用的IP地址
func (ipam *IPAM) Allocate(subnet *net.IPNet) (ip net.IP, err error) {
	// 存放网段中地址分配信息
	ipam.Subnets = &map[string]string{}
	// 从文件中加载已经分配的网段信息
	err = ipam.load()
	if err != nil {
		logrus.Infof("Allocate: error load allocation info, %v", err)
	}

	// 转换IPNet对象
	_, subnet, _ = net.ParseCIDR(subnet.String())

	// 获取子网掩码在长度和网段前面的固定位的长度
	// 如"127.0.0.1/8"网段的子网掩码是"255.0.0.0"
	// 那么subnet.Mask.Size*()的返回值就是前面255对应的位数和总位数,即8和24
	one, size := subnet.Mask.Size()

	// 如果之前没有分配过这个网段,则初始化网段分配配置
	if _, exist := (*ipam.Subnets)[subnet.String()]; !exist {
		(*ipam.Subnets)[subnet.String()] = strings.Repeat("0", 1<<uint8(size-one))
	}

	// 遍历网段的位图数组
	for c := range (*ipam.Subnets)[subnet.String()] {
		// 找到数组的0项和数组序号即可分配IP
		if (*ipam.Subnets)[subnet.String()][c] == '0' {
			// 设置这个'0'项的值位'1',即分配这个IP
			ipalloc := []byte((*ipam.Subnets)[subnet.String()])
			// go的字符串创建后就不能修改,所以通过转换成byte数组,修改后再转化位字符串赋值
			ipalloc[c] = '1'
			(*ipam.Subnets)[subnet.String()] = string(ipalloc)
			// 初始IP
			ip = subnet.IP
			// 通过网段的IP与上面的偏移相加计算出分配的IP地址，由于IP地址是uint的一个数组
			// 需要通过数组中的每一项相加所需的值
			// 如网段"172.16.0.0/12"，数组序号是65555,那么在"172.16.0.0"上依次加
			// [uint8(65555>>24),uint8(65555>>16),uint8(65555>>8), uint8(65555>>0)]
			// 即[0,1,0,19],那么获得的IP为172.17.0.19
			for t := uint(4); t > 0; t -= 1 {
				[]byte(ip)[4-t] += uint8(c >> ((t - 1) * 8))
			}
			// IP从1开始分配,所以此处+1
			ip[3] += 1
			break
		}
	}
	// 将结果保存到文件中
	if err := ipam.dump(); err != nil {
		logrus.Errorf("ipam dump error %v", err)
	}
	return ip, nil
}

// 地址释放
func (ipam *IPAM) Release(subnet *net.IPNet, ipaddr *net.IP) error {
	ipam.Subnets = &map[string]string{}
	_, subnet, _ = net.ParseCIDR(subnet.String())

	// 从文件中加载网段的分配信息
	err := ipam.load()
	if err != nil {
		logrus.Errorf("error dump allocation info, %v", err)
	}

	// 计算IP地址在网段位图中的索引位置
	c := 0
	// 将IP地址转化位4字节的表示方式
	releaseIP := ipaddr.To4()
	// 由于IP是从1开始分配,所以此处-1
	releaseIP[3] -= 1
	for t := uint(4); t > 0; t -= 1 {
		// 与IP分配相反
		c += int(releaseIP[t-1]-subnet.IP[t-1]) << ((4 - t) * 8)
	}

	// 将分配的位图数组中索引位置的值置为0
	ipalloc := []byte((*ipam.Subnets)[subnet.String()])
	ipalloc[c] = '0'
	(*ipam.Subnets)[subnet.String()] = string(ipalloc)

	// 保存
	if err := ipam.dump(); err != nil {
		return err
	}
	return nil
}
