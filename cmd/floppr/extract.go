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
