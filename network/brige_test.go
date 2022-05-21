package network

import (
	"minidocker/container"
	"testing"
)

func TestBridgeInit(t *testing.T) {
	d := BridgeNetworkDriver{}
	_, err := d.Create("192.168.0.1/24", "testbridge")
	t.Logf("err: %v", err)
}

func TestBridgeConnect(t *testing.T) {
	ep := Endpoint{
		ID: "testcontainer",
	}

	n := Network{
		Name: "testbridge",
	}

	d := BridgeNetworkDriver{}
	err := d.Connect(&n, &ep)
	t.Logf("err: %v", err)
}

func TestNetworkConnect(t *testing.T) {
	cInfo := &container.ContainerInfo{
		Id:  "testcontainer",
		Pid: "15438",
	}
	d := BridgeNetworkDriver{}
	n, err := d.Create("192.168.0.1/24", "testbridge")
	t.Logf("error: %v", err)

  if err := Init(); err != nil {
    t.Logf("error init %v", err)
  }

	networks[n.Name] = n
	err = Connect(cInfo.Name, cInfo)
	t.Logf("error: %v", err)
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
