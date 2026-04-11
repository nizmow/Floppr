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
}

func TestRunHelp(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	err := run(context.Background(), nil, &stdout, io.Discard)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if !strings.Contains(stdout.String(), "create") || !strings.Contains(stdout.String(), "Create DOS floppy disk images from directories") {
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

func TestRunCreateHelp(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	err := run(context.Background(), []string{"create", "--help"}, &stdout, io.Discard)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if !strings.Contains(stdout.String(), "floppr create <source> [output] [--format SIZE] [--label LABEL]") {
		t.Fatalf("stdout = %q, want create usage", stdout.String())
	}
	if !strings.Contains(stdout.String(), "--format") {
		t.Fatalf("stdout = %q, want format flag", stdout.String())
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

func TestRunExtractSingleImage(t *testing.T) {
	t.Parallel()

	source := t.TempDir()
	writeFile(t, filepath.Join(source, "README.TXT"), []byte("hello from extract"))
	image := filepath.Join(t.TempDir(), "single.img")

	err := run(context.Background(), []string{"create", source, image}, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("run(create) error = %v", err)
	}

	dest := filepath.Join(t.TempDir(), "out")
	err = run(context.Background(), []string{"extract", image, dest}, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("run(extract) error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dest, "README.TXT"))
	if err != nil {
		t.Fatalf("ReadFile(README.TXT): %v", err)
	}
	if string(data) != "hello from extract" {
		t.Fatalf("README.TXT = %q, want %q", string(data), "hello from extract")
	}
}

func TestRunExtractGlobCreatesPerImageDirectories(t *testing.T) {
	t.Parallel()

	imagesDir := t.TempDir()
	sourceA := filepath.Join(t.TempDir(), "disk-a")
	sourceB := filepath.Join(t.TempDir(), "disk-b")
	writeFile(t, filepath.Join(sourceA, "A.TXT"), []byte("disk a"))
	writeFile(t, filepath.Join(sourceB, "B.TXT"), []byte("disk b"))

	err := run(context.Background(), []string{"create", sourceA, filepath.Join(imagesDir, "disk-a.img")}, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("run(create sourceA) error = %v", err)
	}
	err = run(context.Background(), []string{"create", sourceB, filepath.Join(imagesDir, "disk-b.img")}, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("run(create sourceB) error = %v", err)
	}

	dest := filepath.Join(t.TempDir(), "extract")
	err = run(context.Background(), []string{"extract", filepath.Join(imagesDir, "*.img"), dest}, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("run(extract glob) error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dest, "disk-a", "A.TXT"))
	if err != nil {
		t.Fatalf("ReadFile(disk-a/A.TXT): %v", err)
	}
	if string(data) != "disk a" {
		t.Fatalf("disk-a/A.TXT = %q, want %q", string(data), "disk a")
	}

	data, err = os.ReadFile(filepath.Join(dest, "disk-b", "B.TXT"))
	if err != nil {
		t.Fatalf("ReadFile(disk-b/B.TXT): %v", err)
	}
	if string(data) != "disk b" {
		t.Fatalf("disk-b/B.TXT = %q, want %q", string(data), "disk b")
	}
}

func TestRunExtractMultipleExpandedSourcesCreatesPerImageDirectories(t *testing.T) {
	t.Parallel()

	imagesDir := t.TempDir()
	sourceA := filepath.Join(t.TempDir(), "disk-a")
	sourceB := filepath.Join(t.TempDir(), "disk-b")
	writeFile(t, filepath.Join(sourceA, "A.TXT"), []byte("disk a"))
	writeFile(t, filepath.Join(sourceB, "B.TXT"), []byte("disk b"))

	imageA := filepath.Join(imagesDir, "disk-a.img")
	imageB := filepath.Join(imagesDir, "disk-b.img")

	err := run(context.Background(), []string{"create", sourceA, imageA}, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("run(create sourceA) error = %v", err)
	}
	err = run(context.Background(), []string{"create", sourceB, imageB}, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("run(create sourceB) error = %v", err)
	}

	dest := filepath.Join(t.TempDir(), "extract")
	err = run(context.Background(), []string{"extract", imageA, imageB, dest}, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("run(extract multiple sources) error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dest, "disk-a", "A.TXT"))
	if err != nil {
		t.Fatalf("ReadFile(disk-a/A.TXT): %v", err)
	}
	if string(data) != "disk a" {
		t.Fatalf("disk-a/A.TXT = %q, want %q", string(data), "disk a")
	}

	data, err = os.ReadFile(filepath.Join(dest, "disk-b", "B.TXT"))
	if err != nil {
		t.Fatalf("ReadFile(disk-b/B.TXT): %v", err)
	}
	if string(data) != "disk b" {
		t.Fatalf("disk-b/B.TXT = %q, want %q", string(data), "disk b")
	}
}

func TestRunExtractRequiresSourceAndDestination(t *testing.T) {
	t.Parallel()

	err := run(context.Background(), []string{"extract"}, io.Discard, io.Discard)
	if err == nil {
		t.Fatal("run() error = nil, want missing arguments failure")
	}
	if !strings.Contains(err.Error(), "extract requires one or more <source-or-glob> values and a <destination>") {
		t.Fatalf("run() error = %v, want required args failure", err)
	}
}

func TestRunExtractHelp(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	err := run(context.Background(), []string{"extract", "--help"}, &stdout, io.Discard)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if !strings.Contains(stdout.String(), "floppr extract <source-or-glob>... <destination>") {
		t.Fatalf("stdout = %q, want extract usage", stdout.String())
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
