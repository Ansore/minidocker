package main

import (
	"fmt"
	"minidocker/cgroups/subsystems"
	"minidocker/container"

	"github.com/urfave/cli"
)

var initCommand = cli.Command{
	Name:  "init",
	Usage: "init container process run user's process in container. Do not call it outside",
	Action: func(context *cli.Context) error {
    container.RunContainerInitProcess()
		return nil
	},
}

var runCommand = cli.Command{
	Name:  "run",
	Usage: "create a container with namespace and cgroups limit. minidocker run -it [command]",
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "ti",
			Usage: "enable tty",
		},
    cli.StringFlag{
      Name: "m",
      Usage: "memory limit",
    },
    cli.StringFlag{
      Name: "cpushare",
      Usage: "cpushare limit",
    },
    cli.StringFlag{
      Name: "cpuset",
      Usage: "cpuset limit",
    },
	},
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("Missing container command")
		}
    var cmdArr []string
  for _, arg := range context.Args() {
      cmdArr = append(cmdArr, arg)
    }
    tty := context.Bool("ti")
    resConf := &subsystems.ResourceConfig{
      MemoryLimit: context.String("m"),
      CpuSet: context.String("cpuset"),
      CpuShare: context.String("cpushare"),
    }
    Run(tty, cmdArr, resConf)
		return nil
	},
}
