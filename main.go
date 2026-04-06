package main

import (
	"fmt"
	"os"

	"github.com/alexmaze/clink/internal/cli"
)

var Version = "dev"

func main() {
	cli.Version = Version
	if err := cli.Run(os.Args, os.Stdout, os.Stderr); err != nil {
		if !cli.IsSilent(err) && err.Error() != "" {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}
