package network

import (
	"net"
	"testing"
)

func TestAllocate(t *testing.T) {
	_, ipnet, _ := net.ParseCIDR("192.168.0.1/24")
	ip, _ := ipAllocator.Allocate(ipnet)
	t.Logf("alloc ip1: %v\n", ip)

	_, ipnet, _ = net.ParseCIDR("192.168.0.2/24")
	ip, _ = ipAllocator.Allocate(ipnet)
	t.Logf("alloc ip2: %v\n", ip)

	_, ipnet, _ = net.ParseCIDR("192.168.0.3/24")
	ip, _ = ipAllocator.Allocate(ipnet)
	t.Logf("alloc ip2: %v\n", ip)
}

func TestRelease(t *testing.T) {
	ip, ipnet, _ := net.ParseCIDR("192.168.0.1/24")
	if err := ipAllocator.Release(ipnet, &ip); err != nil {
		t.Logf("error %v\n", err)
	}
	ip, ipnet, _ = net.ParseCIDR("192.168.0.2/24")
	if err := ipAllocator.Release(ipnet, &ip); err != nil {
		t.Logf("error %v\n", err)
	}
	ip, ipnet, _ = net.ParseCIDR("192.168.0.3/24")
	if err := ipAllocator.Release(ipnet, &ip); err != nil {
		t.Logf("error %v\n", err)
	}
}
