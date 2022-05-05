package subsystems

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
)

type CpusetSubSystem struct {}

func (s *CpusetSubSystem) Name() string {
	return "cpuset"
}

func (s *CpusetSubSystem) Set(cgroupPath string, res *ResourceConfig) error {
  if subsysCgroupPath, err := GetCgroupPath(s.Name(), cgroupPath, true); err == nil {
    if res.CpuShare != "" {
      if err := ioutil.WriteFile(path.Join(subsysCgroupPath, "cpuset.cpus"),
        []byte(res.CpuSet), 0644); err != nil {
        return fmt.Errorf("set cgroup cpuset fail %v", err)
      }
    }
  } else {
    return err
  }
  return nil
}

func (s *CpusetSubSystem) Remove(cgroupPath string) error {
  if subsysCgroupPath, err := GetCgroupPath(s.Name(), cgroupPath, false); err == nil {
    return os.RemoveAll(subsysCgroupPath)
  }
  return nil
}

func (s *CpusetSubSystem) Apply(cgroupPath string, pid int) error {
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
