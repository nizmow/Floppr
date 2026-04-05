package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

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
	if len(args) == 0 {
		_, err := io.WriteString(stdout, rootUsage())
		return err
	}

	switch args[0] {
	case "create":
		return runCreate(ctx, args[1:], stderr)
	case "help", "-h", "--help":
		_, err := io.WriteString(stdout, rootUsage())
		return err
	case "version", "--version":
		_, err := fmt.Fprintf(stdout, "floppr %s\n", version)
		return err
	default:
		return fmt.Errorf("unknown command %q\n\n%s", args[0], rootUsage())
	}
}

func runCreate(ctx context.Context, args []string, stderr io.Writer) error {
	fs := flag.NewFlagSet("floppr create", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		_, _ = io.WriteString(stderr, createUsage())
	}

	label := fs.String("label", "", "DOS volume label")
	fs.StringVar(label, "l", "", "DOS volume label")
	format := fs.String("format", floppy.DefaultFormat(), "Floppy format in KB")
	fs.StringVar(format, "f", floppy.DefaultFormat(), "Floppy format in KB")

	normalizedArgs, err := normalizeCreateArgs(args)
	if err != nil {
		fs.Usage()
		return err
	}

	if err := fs.Parse(normalizedArgs); err != nil {
		return err
	}

	positionals := fs.Args()
	if len(positionals) < 1 || len(positionals) > 2 {
		fs.Usage()
		return errors.New("create requires <source> and optional [output]")
	}

	return floppy.CreateImage(ctx, floppy.Options{
		SourceDir:   positionals[0],
		OutputPath:  defaultOutputPath(positionals),
		VolumeLabel: defaultVolumeLabel(positionals[0], *label),
		Format:      *format,
	})
}

func rootUsage() string {
	return strings.TrimLeft(`
Floppr builds DOS floppy disk images from directories.

Usage:
  floppr create <source> [output] [--format SIZE] [--label LABEL]
  floppr help
  floppr version

Examples:
  floppr create ./MYGAME ./MYGAME.img --label MYGAME
  floppr create ./disk-contents
  floppr create ./disk-contents --format 720
`, "\n")
}

func createUsage() string {
	return strings.TrimLeft(`
Create a DOS 1.44MB floppy image from a directory.

Usage:
  floppr create <source> [output] [--format SIZE] [--label LABEL]
  floppr create [--format SIZE] [--label LABEL] <source> [output]

Arguments:
  source    Directory to package into the floppy image
  output    Optional path to the output .img file

Options:
  --format  Floppy size in KB: 360, 720, 1200, 1440 (default: 1440)
  -f        Short form of --format
  --label   DOS volume label (defaults from directory name)
  -l        Short form of --label
`, "\n")
}

func normalizeCreateArgs(args []string) ([]string, error) {
	var (
		flags       []string
		positionals []string
	)

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--":
			positionals = append(positionals, args[i+1:]...)
			i = len(args)
		case arg == "--label" || arg == "-l" || arg == "--format" || arg == "-f":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("%s requires a value", arg)
			}
			flags = append(flags, arg, args[i+1])
			i++
		case strings.HasPrefix(arg, "--label="):
			flags = append(flags, arg)
		case strings.HasPrefix(arg, "-l="):
			flags = append(flags, arg)
		case strings.HasPrefix(arg, "--format="):
			flags = append(flags, arg)
		case strings.HasPrefix(arg, "-f="):
			flags = append(flags, arg)
		case strings.HasPrefix(arg, "-"):
			flags = append(flags, arg)
		default:
			positionals = append(positionals, arg)
		}
	}

	return append(flags, positionals...), nil
}

func defaultOutputPath(positionals []string) string {
	if len(positionals) == 2 {
		return positionals[1]
	}
	return floppy.DefaultOutputPath(positionals[0])
}

func defaultVolumeLabel(sourceDir, label string) string {
	if label != "" {
		return label
	}
	return floppy.DefaultVolumeLabel(sourceDir)
}
