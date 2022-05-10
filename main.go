package main

import (
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
		initCommand,
    runCommand,
    commitCommand,
    listCommand,
    logCommand,
	}
	app.Before = func(ctx *cli.Context) error {
		log.SetFormatter(&log.JSONFormatter{})
		log.SetOutput(os.Stdout)
		return nil
	}
  if err := app.Run(os.Args); err != nil {
    log.Fatal(err)
  }
}
