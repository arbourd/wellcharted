package main

import (
	"log"
	"os"

	"github.com/arbourd/wellcharted/cmd"
	"github.com/mitchellh/cli"
)

func main() {
	ui := &cli.BasicUi{
		Reader:      os.Stdin,
		Writer:      os.Stdout,
		ErrorWriter: os.Stderr,
	}

	c := cli.NewCLI("wellcharted", "0.0.1")
	c.Args = os.Args[1:]
	c.Commands = map[string]cli.CommandFactory{
		"semver bump": func() (cli.Command, error) {
			return &cmd.Bump{UI: ui}, nil
		},
		"semver compare": func() (cli.Command, error) {
			return &cmd.Compare{UI: ui}, nil
		},
	}

	exitStatus, err := c.Run()
	if err != nil {
		log.Println(err)
	}

	os.Exit(exitStatus)
}
