package main

import (
	"os"

	"github.com/TyrellD1/agenttohuman/internal/cli"
)

func main() {
	args := os.Args[1:]
	if err := cli.Run(args, os.Stdin, os.Stdout, os.Stderr); err != nil {
		cli.WriteError(os.Stderr, cli.HasJSON(args), err)
		os.Exit(1)
	}
}
