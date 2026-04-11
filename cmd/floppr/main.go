package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	cli "github.com/urfave/cli/v3"
)

const version = "0.1.0"

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if err := validateRootArgs(args); err != nil {
		return err
	}
	return newCommand(stdout, stderr).Run(ctx, append([]string{"floppr"}, args...))
}

func newCommand(stdout, stderr io.Writer) *cli.Command {
	return &cli.Command{
		Name:      "floppr",
		Usage:     "Create DOS floppy disk images from directories",
		Version:   version,
		Writer:    stdout,
		ErrWriter: stderr,
		OnUsageError: func(_ context.Context, _ *cli.Command, err error, _ bool) error {
			return err
		},
		Commands: []*cli.Command{
			newCreateCommand(stderr),
			newExtractCommand(stderr),
			newVersionCommand(stdout),
		},
	}
}

func validateRootArgs(args []string) error {
	if len(args) == 0 {
		return nil
	}
	if strings.HasPrefix(args[0], "-") {
		return nil
	}
	switch args[0] {
	case "create", "extract", "help", "version":
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}
