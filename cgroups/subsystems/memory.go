package subsystems

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
)

type MemorySubSystem struct {
	used bool
}

func (s *MemorySubSystem) Name() string {
	return "memory"
}

func (s *MemorySubSystem) Set(cgroupPath string, res *ResourceConfig) error {
	if subsysCgroupPath, err := GetCgroupPath(s.Name(), cgroupPath, true); err == nil {
		if res.MemoryLimit != "" {
			if err := ioutil.WriteFile(path.Join(subsysCgroupPath, "memory.limit_in_bytes"),
				[]byte(res.MemoryLimit), 0644); err != nil {
				return fmt.Errorf("set cgroup memory failed %v", err)
			}
		  s.used = true
		}
		return nil
	} else {
		return err
	}
}

func (s *MemorySubSystem) Remove(cgroupPath string) error {
	if s.used {
		if subsysCgroupPath, err := GetCgroupPath(s.Name(), cgroupPath, false); err == nil {
			return os.RemoveAll(subsysCgroupPath)
		} else {
			return err
		}
	}
	return nil
}

func (s *MemorySubSystem) Apply(cgroupPath string, pid int) error {
	if s.used {
		if subsysCgroupPath, err := GetCgroupPath(s.Name(), cgroupPath, false); err == nil {
			if err := ioutil.WriteFile(path.Join(subsysCgroupPath, "tasks"),
				[]byte(strconv.Itoa(pid)), 0644); err != nil {
				return fmt.Errorf("set cgroup %s proc failed %v", s.Name(), err)
			}
		} else {
			return fmt.Errorf("get cgroup %s error: %v", cgroupPath, err)
		}
	}
	return nil
}
