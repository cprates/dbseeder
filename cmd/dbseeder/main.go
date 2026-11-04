package main

import (
	"context"
	"os"

	"github.com/urfave/cli/v3"
)

func main() {
	if err := buildCMDs().Run(context.Background(), os.Args); err != nil {
		os.Exit(1)
	}
}

func buildCMDs() *cli.Command {
	return &cli.Command{
		Name:  "dbseeder",
		Usage: "CLI tool to seed databases",
		Commands: []*cli.Command{
			seedCommand(),
		},
	}
}
