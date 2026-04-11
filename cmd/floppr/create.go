package main

import (
	"context"
	"fmt"
	"io"

	cli "github.com/urfave/cli/v3"

	"github.com/nizmow/floppr/internal/floppy"
)

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
