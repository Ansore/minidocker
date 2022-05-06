package main

import (
	// "minidocker/cgroups"
	"minidocker/cgroups/subsystems"
	"minidocker/container"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

func sendInitCommand(cmdArr []string, writePipe *os.File) {
  command := strings.Join(cmdArr, " ")
  logrus.Infof("command all is %s", command)
  writePipe.WriteString(command)
  writePipe.Close()
}

func Run(tty bool, cmdArr []string, resConf *subsystems.ResourceConfig, volume string) {
  parent, writePipe := container.NewParentProcess(tty, volume)
  if parent == nil {
    logrus.Errorf("New parent process error")
    return
  }
  if err := parent.Start(); err != nil {
    logrus.Error(err)
  }
  // cgroupManager := cgroups.NewCgroupManager("minidocker-cgroup")
  // defer cgroupManager.Destroy()
  // cgroupManager.Set(resConf)
  // cgroupManager.Apply(parent.Process.Pid)
  sendInitCommand(cmdArr, writePipe)
  parent.Wait()
  mntURL := "/root/mnt"
  rootURL := "/root/"
  container.DeleteWorkSpace(rootURL, mntURL, volume)
  os.Exit(0)
}
