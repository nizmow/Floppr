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
		printRootUsage(stdout)
		return nil
	}

	switch args[0] {
	case "create":
		return runCreate(ctx, args[1:], stderr)
	case "help", "-h", "--help":
		printRootUsage(stdout)
		return nil
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

	label := fs.String("label", "FLOPPR", "DOS volume label")
	fs.StringVar(label, "l", "FLOPPR", "DOS volume label")

	normalizedArgs, err := normalizeCreateArgs(args)
	if err != nil {
		fs.Usage()
		return err
	}

	if err := fs.Parse(normalizedArgs); err != nil {
		return err
	}

	positionals := fs.Args()
	if len(positionals) != 2 {
		fs.Usage()
		return errors.New("create requires <source> and <output>")
	}

	opts := floppy.Options{
		SourceDir:   positionals[0],
		OutputPath:  positionals[1],
		VolumeLabel: *label,
	}

	return floppy.CreateImage(ctx, opts)
}

func printRootUsage(w io.Writer) {
	_, _ = io.WriteString(w, rootUsage())
}

func rootUsage() string {
	return strings.TrimLeft(`
Floppr builds DOS 1.44MB floppy disk images from directories.

Usage:
  floppr create <source> <output> [--label LABEL]
  floppr help
  floppr version

Examples:
  floppr create ./MYGAME ./MYGAME.img --label MYGAME
  floppr create ./disk-contents ./bootdisk.img
`, "\n")
}

func createUsage() string {
	return strings.TrimLeft(`
Create a DOS 1.44MB floppy image from a directory.

Usage:
  floppr create <source> <output> [--label LABEL]
  floppr create [--label LABEL] <source> <output>

Arguments:
  source    Directory to package into the floppy image
  output    Path to the output .img file

Options:
  --label   DOS volume label (default: FLOPPR)
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
		case arg == "--label" || arg == "-l":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("%s requires a value", arg)
			}
			flags = append(flags, arg, args[i+1])
			i++
		case strings.HasPrefix(arg, "--label="):
			flags = append(flags, arg)
		case strings.HasPrefix(arg, "-l="):
			flags = append(flags, arg)
		case strings.HasPrefix(arg, "-"):
			flags = append(flags, arg)
		default:
			positionals = append(positionals, arg)
		}
	}

	return append(flags, positionals...), nil
}
