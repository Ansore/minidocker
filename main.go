package main

import (
	cmd "minidocker/command"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

const usage = `minidocker is a simple container runtime implementation.`

func main() {
	app := cli.NewApp()
	app.Name = "minidocker"
	app.Usage = usage

	app.Commands = []cli.Command{
		cmd.InitCommand,
    cmd.RunCommand,
    cmd.CommitCommand,
    cmd.ListCommand,
    cmd.LogCommand,
    cmd.ExecCommand,
    cmd.StopCommand,
	}
	app.Before = func(_ *cli.Context) error {
		log.SetFormatter(&log.JSONFormatter{})
		log.SetOutput(os.Stdout)
		return nil
	}
  if err := app.Run(os.Args); err != nil {
    log.Fatal(err)
  }
}
