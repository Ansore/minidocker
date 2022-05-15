package command

import (
	"fmt"
	"minidocker/cgroups/subsystems"
	"minidocker/container"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var InitCommand = cli.Command{
	Name:  "init",
	Usage: "init container process run user's process in container. Do not call it outside",
	Action: func(_ *cli.Context) error {
		if err := container.RunContainerInitProcess(); err != nil {
			logrus.Infof("init failed!")
		}
		return nil
	},
}

var RunCommand = cli.Command{
	Name:  "run",
	Usage: "create a container with namespace and cgroups limit. minidocker run -it [command]",
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "ti",
			Usage: "enable tty",
		},
		cli.BoolFlag{
			Name:  "d",
			Usage: "detach container",
		},
		cli.StringFlag{
			Name:  "m",
			Usage: "memory limit",
		},
		cli.StringFlag{
			Name:  "cpushare",
			Usage: "cpushare limit",
		},
		cli.StringFlag{
			Name:  "cpuset",
			Usage: "cpuset limit",
		},
		cli.StringFlag{
			Name:  "v",
			Usage: "volume",
		},
		cli.StringFlag{
			Name:  "name",
			Usage: "container name",
		},
	},
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("missing container command")
		}
		var cmdArr []string
		for _, arg := range context.Args() {
			cmdArr = append(cmdArr, arg)
		}
		resConf := &subsystems.ResourceConfig{
			MemoryLimit: context.String("m"),
			CpuSet:      context.String("cpuset"),
			CpuShare:    context.String("cpushare"),
		}

		volume := context.String("v")

		// tty 与 detach 不能共存
		createTty := context.Bool("ti")
		detach := context.Bool("d")

		if createTty && detach {
			return fmt.Errorf("ti and d paramter can not both provided")
		}
		containerName := context.String("name")
		Run(createTty, cmdArr, resConf, volume, containerName)
		return nil
	},
}

var CommitCommand = cli.Command{
	Name:  "commit",
	Usage: "commit a container into image",
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("missing container name")
		}
		imageName := context.Args().Get(0)
		// commitContainer(containerName)
		commitContainer(imageName)
		return nil
	},
}

var ListCommand = cli.Command{
	Name:  "ps",
	Usage: "list all the containers",
	Action: func(_ *cli.Context) error {
		ListContainers()
		return nil
	},
}

var LogCommand = cli.Command{
	Name:  "logs",
	Usage: "print logs of a container",
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("please input your container name")
		}
		containerName := context.Args().Get(0)
		// commitContainer(containerName)
		logContainer(containerName)
		return nil
	},
}

var ExecCommand = cli.Command{
	Name:  "exec",
	Usage: "exec a command into container",
	Action: func(context *cli.Context) error {
		// this is for callback
		if os.Getenv(ENV_EXEC_PID) != "" {
			logrus.Infof("pid callback pid %d", os.Getgid())
			return nil
		}
		if len(context.Args()) < 2 {
			return fmt.Errorf("missing container name or command")
		}
		containerName := context.Args().Get(0)
		var cmdArray []string
		// 除了容器名之外的参数作为需要执行的命令处理
		cmdArray = append(cmdArray, context.Args().Tail()...)
		// 执行命令
		ExecContainer(containerName, cmdArray)
		return nil
	},
}

var StopCommand = cli.Command {
	Name:  "stop",
	Usage: "stop a container",
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("missing container name")
		}
    containerName := context.Args().Get(0)
    stopContainer(containerName)
		return nil
	},
}

var RemoveCommand = cli.Command {
	Name:  "rm",
	Usage: "remove a container",
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("missing container name")
		}
    containerName := context.Args().Get(0)
    removeContainer(containerName)
		return nil
	},
}
