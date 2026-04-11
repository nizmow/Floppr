package main

import (
	"context"
	"fmt"
	"io"

	cli "github.com/urfave/cli/v3"
)

func newVersionCommand(stdout io.Writer) *cli.Command {
	return &cli.Command{
		Name:   "version",
		Usage:  "Print the version",
		Writer: stdout,
		Action: func(_ context.Context, cmd *cli.Command) error {
			_, err := fmt.Fprintf(cmd.Writer, "floppr %s\n", version)
			return err
		},
	}
}
