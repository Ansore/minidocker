package network

import (
	"minidocker/container"
	"testing"
)

func TestBridgeInit(t *testing.T) {
	d := BridgeNetworkDriver{}
	nw, err := d.Create("192.168.0.1/24", "testbridge")
	t.Logf("create err: %v", err)
	err = d.Delete(*nw)
	t.Logf("delete err: %v", err)
}

func TestBridgeConnect(t *testing.T) {
	d := BridgeNetworkDriver{}
	nw, err := d.Create("192.168.0.1/24", "testbridge")
	t.Logf("create err: %v", err)

	ep := Endpoint{
		ID: "testcontainer",
	}

	n := Network{
		Name: "testbridge",
	}

	err = d.Connect(&n, &ep)
	t.Logf("err: %v", err)
	err = d.Delete(*nw)
	t.Logf("delete err: %v", err)
}

func TestNetworkConnect(t *testing.T) {
	cInfo := &container.ContainerInfo{
		Id:  "testcontainer",
		Pid: "15438",
	}
	d := BridgeNetworkDriver{}
	n, err := d.Create("192.168.0.1/24", "testbridge")
	t.Logf("create error: %v", err)

	if err := Init(); err != nil {
		t.Logf("init error %v", err)
	}

	networks[n.Name] = n
	err = Connect(cInfo.Name, cInfo)
	t.Logf("connect error: %v", err)
	err = d.Delete(*n)
	t.Logf("delete err: %v", err)
}

func TestLoad(t *testing.T) {
	n := Network{
		Name: "testbridge",
	}
	if err := n.load("/var/run/minidocker/network/network/testbridge"); err != nil {
		t.Logf("error load %v", err)
	}
	t.Logf("network: %v", n)
}
