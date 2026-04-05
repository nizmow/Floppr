package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	diskfs "github.com/diskfs/go-diskfs"
)

func TestRunCreateCreatesDiskImage(t *testing.T) {
	t.Parallel()

	source := t.TempDir()
	writeFile(t, filepath.Join(source, "README.TXT"), []byte("hello from cli"))
	writeFile(t, filepath.Join(source, "BIN", "RUN.EXE"), []byte("run"))

	output := filepath.Join(t.TempDir(), "cli.img")
	err := run(context.Background(), []string{
		"create", source, output, "--label", "CLITEST", "--format", "1440",
	}, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}

	d, err := diskfs.Open(output)
	if err != nil {
		t.Fatalf("diskfs.Open(): %v", err)
	}
	defer d.Close()

	imgFS, err := d.GetFilesystem(0)
	if err != nil {
		t.Fatalf("GetFilesystem(0): %v", err)
	}
	defer imgFS.Close()

	data, err := imgFS.ReadFile("/README.TXT")
	if err != nil {
		t.Fatalf("ReadFile(/README.TXT): %v", err)
	}
	if string(data) != "hello from cli" {
		t.Fatalf("ReadFile(/README.TXT) = %q", string(data))
	}
}

func TestRunCreateFailsForOversizePayload(t *testing.T) {
	t.Parallel()

	source := t.TempDir()
	writeFile(t, filepath.Join(source, "BIG.BIN"), make([]byte, 1_457_665))

	err := run(context.Background(), []string{
		"create", source, filepath.Join(t.TempDir(), "cli.img"), "--label", "TOOBIG", "--format", "1440",
	}, io.Discard, io.Discard)
	if err == nil {
		t.Fatal("run() error = nil, want oversize failure")
	}
	if !strings.Contains(err.Error(), "available on a 1.44MB floppy") {
		t.Fatalf("run() error = %v, want capacity failure", err)
	}
}

func TestRunCreateRequiresSourceAndOutput(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	err := run(context.Background(), []string{"create"}, io.Discard, &stderr)
	if err == nil {
		t.Fatal("run() error = nil, want missing arguments failure")
	}
	if !strings.Contains(err.Error(), "create requires <source> and optional [output]") {
		t.Fatalf("run() error = %v, want required args failure", err)
	}
	if !strings.Contains(stderr.String(), "Usage:\n  floppr create <source> [output] [--format SIZE] [--label LABEL]") {
		t.Fatalf("stderr = %q, want create usage", stderr.String())
	}
}

func TestRunHelp(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	err := run(context.Background(), []string{"help"}, &stdout, io.Discard)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if !strings.Contains(stdout.String(), "floppr create <source> [output] [--format SIZE] [--label LABEL]") {
		t.Fatalf("stdout = %q, want root usage", stdout.String())
	}
}

func TestRunVersion(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	err := run(context.Background(), []string{"version"}, &stdout, io.Discard)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if strings.TrimSpace(stdout.String()) != "floppr "+version {
		t.Fatalf("stdout = %q, want version", stdout.String())
	}
}

func TestRunRejectsUnknownCommand(t *testing.T) {
	t.Parallel()

	err := run(context.Background(), []string{"wat"}, io.Discard, io.Discard)
	if err == nil {
		t.Fatal("run() error = nil, want unknown command failure")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("run() error = %v, want unknown command failure", err)
	}
}

func TestRunCreateDefaultsOutputAndLabel(t *testing.T) {
	t.Parallel()

	parent := t.TempDir()
	source := filepath.Join(parent, "My Great Game Disk")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", source, err)
	}
	writeFile(t, filepath.Join(source, "README.TXT"), []byte("auto defaults"))

	err := run(context.Background(), []string{"create", source}, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}

	output := filepath.Join(parent, "My Great Game Disk.img")
	if _, err := os.Stat(output); err != nil {
		t.Fatalf("Stat(%q): %v", output, err)
	}

	d, err := diskfs.Open(output)
	if err != nil {
		t.Fatalf("diskfs.Open(): %v", err)
	}
	defer d.Close()

	imgFS, err := d.GetFilesystem(0)
	if err != nil {
		t.Fatalf("GetFilesystem(0): %v", err)
	}
	defer imgFS.Close()

	if got := strings.TrimSpace(imgFS.Label()); got != "MYGREATGAME" {
		t.Fatalf("Label() = %q, want %q", got, "MYGREATGAME")
	}
}

func TestRunCreateSupportsFormatFlag(t *testing.T) {
	t.Parallel()

	source := t.TempDir()
	writeFile(t, filepath.Join(source, "README.TXT"), []byte("720k"))
	output := filepath.Join(t.TempDir(), "cli-720.img")

	err := run(context.Background(), []string{"create", source, output, "--format", "720"}, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}

	info, err := os.Stat(output)
	if err != nil {
		t.Fatalf("Stat(%q): %v", output, err)
	}
	if info.Size() != 737280 {
		t.Fatalf("image size = %d, want %d", info.Size(), 737280)
	}
}

func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}
