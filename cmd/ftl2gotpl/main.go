package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/cruffinoni/ftl2gotpl/internal/cli"
	"github.com/cruffinoni/ftl2gotpl/internal/logging"
)

func main() {
	logging.Configure()

	cmd := cli.NewRootCmd()
	if err := cmd.Execute(); err != nil {
		var exitErr *cli.ExitError
		if errors.As(err, &exitErr) {
			if exitErr.Err != nil {
				fmt.Fprintln(os.Stderr, exitErr.Err)
			}
			os.Exit(exitErr.Code)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
