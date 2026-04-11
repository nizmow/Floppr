package main

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"

	cli "github.com/urfave/cli/v3"

	"github.com/nizmow/floppr/internal/floppy"
)

func newExtractCommand(stderr io.Writer) *cli.Command {
	return &cli.Command{
		Name:      "extract",
		Usage:     "Extract files from one or more floppy images",
		UsageText: "floppr extract <source-or-glob>... <destination>",
		ArgsUsage: "<source-or-glob>... <destination>",
		ErrWriter: stderr,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "flat",
				Usage: "Extract multiple images directly into the destination without per-image subdirectories",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() < 2 {
				return fmt.Errorf("extract requires one or more <source-or-glob> values and a <destination>")
			}

			sources, err := collectSources(cmd.Args().Slice()[:cmd.Args().Len()-1])
			if err != nil {
				return err
			}

			destinations := extractionDestinations(sources, cmd.Args().Get(cmd.Args().Len()-1), cmd.Bool("flat"))
			for i, source := range sources {
				if err := floppy.ExtractImage(ctx, source, destinations[i]); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func collectSources(inputs []string) ([]string, error) {
	var sources []string
	for _, input := range inputs {
		expanded, err := expandSources(input)
		if err != nil {
			return nil, err
		}
		sources = append(sources, expanded...)
	}
	return sources, nil
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

func extractionDestinations(sources []string, destination string, flat bool) []string {
	if len(sources) <= 1 || flat {
		results := make([]string, len(sources))
		for i := range sources {
			results[i] = destination
		}
		return results
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
