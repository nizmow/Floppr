package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	cli "github.com/urfave/cli/v3"

	"github.com/nizmow/floppr/internal/floppy"
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
			newVersionCommand(stdout),
		},
	}
}

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

func newCreateCommand(stderr io.Writer) *cli.Command {
	return &cli.Command{
		Name:      "create",
		Usage:     "Create a DOS floppy image from a directory",
		UsageText: "floppr create <source> [output] [--format SIZE] [--label LABEL]",
		ArgsUsage: "<source> [output]",
		ErrWriter: stderr,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "format",
				Aliases: []string{"f"},
				Usage:   "Floppy size in KB: 360, 720, 1200, 1440",
				Value:   floppy.DefaultFormat(),
			},
			&cli.StringFlag{
				Name:    "label",
				Aliases: []string{"l"},
				Usage:   "DOS volume label",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() < 1 || cmd.Args().Len() > 2 {
				return fmt.Errorf("create requires <source> and optional [output]")
			}

			sourceDir := cmd.Args().Get(0)
			return floppy.CreateImage(ctx, floppy.Options{
				SourceDir:   sourceDir,
				OutputPath:  outputPathFor(sourceDir, cmd.Args().Get(1)),
				VolumeLabel: volumeLabelFor(sourceDir, cmd.String("label")),
				Format:      cmd.String("format"),
			})
		},
	}
}

func outputPathFor(sourceDir, output string) string {
	if output != "" {
		return output
	}
	return floppy.DefaultOutputPath(sourceDir)
}

func volumeLabelFor(sourceDir, label string) string {
	if label != "" {
		return label
	}
	return floppy.DefaultVolumeLabel(sourceDir)
}

func validateRootArgs(args []string) error {
	if len(args) == 0 {
		return nil
	}
	if strings.HasPrefix(args[0], "-") {
		return nil
	}
	switch args[0] {
	case "create", "help", "version":
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}
