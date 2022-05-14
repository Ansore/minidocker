package cgroups

import (
	"minidocker/cgroups/subsystems"

	"github.com/sirupsen/logrus"
)

type CgroupManager struct {
  Path string
  Resource *subsystems.ResourceConfig
}

func NewCgroupManager(path string) *CgroupManager {
  return &CgroupManager{
    Path: path,
  }
}

// pid join the cgroup
func (c *CgroupManager) Apply(pid int) error {
  for _, subSysIns := range(subsystems.SubsystemsIns) {
    if err := subSysIns.Apply(c.Path, pid); err != nil {
      return err
    }
  }
  return nil
}

// set cgroup rule
func (c *CgroupManager) Set(res *subsystems.ResourceConfig) error {
  for _, subSysIns := range(subsystems.SubsystemsIns) {
    if err := subSysIns.Set(c.Path, res); err != nil {
      return err
    }
  }
  return nil
}

// destroy cgroup
func (c *CgroupManager) Destroy() {
  for _, subSysIns := range(subsystems.SubsystemsIns) {
    if err := subSysIns.Remove(c.Path); err != nil {
      logrus.Warnf("remove cgroup fail %v", err)
    }
  }
}
