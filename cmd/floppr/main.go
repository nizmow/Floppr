package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
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
			newExtractCommand(stderr),
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

func newExtractCommand(stderr io.Writer) *cli.Command {
	return &cli.Command{
		Name:      "extract",
		Usage:     "Extract files from one or more floppy images",
		UsageText: "floppr extract <source-or-glob> <destination>",
		ArgsUsage: "<source-or-glob> <destination>",
		ErrWriter: stderr,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() != 2 {
				return fmt.Errorf("extract requires <source-or-glob> and <destination>")
			}

			sources, err := expandSources(cmd.Args().Get(0))
			if err != nil {
				return err
			}

			destinations := extractionDestinations(sources, cmd.Args().Get(1))
			for i, source := range sources {
				if err := floppy.ExtractImage(ctx, source, destinations[i]); err != nil {
					return err
				}
			}
			return nil
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
	case "create", "extract", "help", "version":
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func expandSources(source string) ([]string, error) {
	if !hasGlob(source) {
		return []string{source}, nil
	}

	matches, err := filepath.Glob(source)
	if err != nil {
		return nil, fmt.Errorf("invalid source glob %q: %w", source, err)
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("source glob %q matched no files", source)
	}
	slices.Sort(matches)
	return matches, nil
}

func extractionDestinations(sources []string, destination string) []string {
	if len(sources) <= 1 {
		return []string{destination}
	}

	results := make([]string, len(sources))
	used := make(map[string]int, len(sources))
	for i, source := range sources {
		name := imageBaseName(source)
		used[name]++
		if used[name] > 1 {
			name = fmt.Sprintf("%s-%d", name, used[name])
		}
		results[i] = filepath.Join(destination, name)
	}
	return results
}

func imageBaseName(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	if ext == "" {
		return base
	}
	return strings.TrimSuffix(base, ext)
}

func hasGlob(value string) bool {
	return strings.ContainsAny(value, "*?[")
}
