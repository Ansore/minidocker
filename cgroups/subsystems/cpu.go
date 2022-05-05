package subsystems

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
)

type CpuSubSystem struct {}

func (s *CpuSubSystem) Name() string {
	return "cpu"
}

func (s *CpuSubSystem) Set(cgroupPath string, res *ResourceConfig) error {
  if subsysCgroupPath, err := GetCgroupPath(s.Name(), cgroupPath, true); err == nil {
    if res.CpuShare != "" {
      if err := ioutil.WriteFile(path.Join(subsysCgroupPath, "cpu.shares"),
        []byte(res.CpuShare),0644); err != nil {
        return fmt.Errorf("set cgroup cpu share fail %v", err)
      }
    }
  } else {
    return err
  }
  return nil
}

func (s *CpuSubSystem) Remove(cgroupPath string) error {
  if subsysCgroupPath, err := GetCgroupPath(s.Name(), cgroupPath, false); err == nil {
    return os.RemoveAll(subsysCgroupPath)
  }
  return nil
}

func (s *CpuSubSystem) Apply(cgroupPath string, pid int) error {
  if subsysCgroupPath, err := GetCgroupPath(s.Name(), cgroupPath, true); err == nil {
      if err := ioutil.WriteFile(path.Join(subsysCgroupPath, "tasks"),
        []byte(strconv.Itoa(pid)),0644); err != nil {
        return fmt.Errorf("set cgroup cpu proc fail %v", err)
      }
    return nil
  } else {
    return fmt.Errorf("get cgroup %s error: %v", cgroupPath, err)
  }
}
