package floppy

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	diskfs "github.com/diskfs/go-diskfs"
)

func TestCreateImage(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "README.TXT"), []byte("hello floppy"))
	mustMkdir(t, filepath.Join(root, "BIN"))
	writeFile(t, filepath.Join(root, "BIN", "RUN.EXE"), []byte("binary"))

	output := filepath.Join(t.TempDir(), "disk.img")
	err := CreateImage(context.Background(), Options{
		SourceDir:   root,
		OutputPath:  output,
		VolumeLabel: "TESTDISK",
		Format:      "1440",
	})
	if err != nil {
		t.Fatalf("CreateImage() error = %v", err)
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
	if string(data) != "hello floppy" {
		t.Fatalf("ReadFile(/README.TXT) = %q", string(data))
	}

	data, err = imgFS.ReadFile("/BIN/RUN.EXE")
	if err != nil {
		t.Fatalf("ReadFile(/BIN/RUN.EXE): %v", err)
	}
	if string(data) != "binary" {
		t.Fatalf("ReadFile(/BIN/RUN.EXE) = %q", string(data))
	}
}

func TestCreateImageFailsForOversizePayload(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	profile, err := ParseFormat("1440")
	if err != nil {
		t.Fatalf("ParseFormat(1440): %v", err)
	}
	writeFile(t, filepath.Join(root, "BIG.BIN"), make([]byte, profile.DataAreaBytes+1))

	err = CreateImage(context.Background(), Options{
		SourceDir:   root,
		OutputPath:  filepath.Join(t.TempDir(), "disk.img"),
		VolumeLabel: "BIGDISK",
		Format:      "1440",
	})
	if err == nil {
		t.Fatal("CreateImage() error = nil, want oversize failure")
	}
	if !strings.Contains(err.Error(), "1.44MB") {
		t.Fatalf("CreateImage() error = %v, want capacity message", err)
	}
}

func TestCreateImageRejectsInvalidDOSNames(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "longfilename.txt"), []byte("bad"))

	err := CreateImage(context.Background(), Options{
		SourceDir:  root,
		OutputPath: filepath.Join(t.TempDir(), "disk.img"),
		Format:     DefaultFormat(),
	})
	if err == nil {
		t.Fatal("CreateImage() error = nil, want DOS filename validation failure")
	}
	if !strings.Contains(err.Error(), "DOS") {
		t.Fatalf("CreateImage() error = %v, want DOS validation message", err)
	}
}

func TestCreateImageSupportsDifferentFormats(t *testing.T) {
	t.Parallel()

	profile, err := ParseFormat("720")
	if err != nil {
		t.Fatalf("ParseFormat(720): %v", err)
	}

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "README.TXT"), []byte("small disk"))
	output := filepath.Join(t.TempDir(), "disk.img")

	err = CreateImage(context.Background(), Options{
		SourceDir:   root,
		OutputPath:  output,
		VolumeLabel: "SMALLDISK",
		Format:      "720",
	})
	if err != nil {
		t.Fatalf("CreateImage() error = %v", err)
	}

	info, err := os.Stat(output)
	if err != nil {
		t.Fatalf("Stat(%q): %v", output, err)
	}
	if info.Size() != profile.ImageBytes {
		t.Fatalf("image size = %d, want %d", info.Size(), profile.ImageBytes)
	}
}

func TestExtractImage(t *testing.T) {
	t.Parallel()

	source := t.TempDir()
	writeFile(t, filepath.Join(source, "README.TXT"), []byte("hello floppy"))
	mustMkdir(t, filepath.Join(source, "BIN"))
	writeFile(t, filepath.Join(source, "BIN", "RUN.EXE"), []byte("binary"))

	image := filepath.Join(t.TempDir(), "disk.img")
	err := CreateImage(context.Background(), Options{
		SourceDir:   source,
		OutputPath:  image,
		VolumeLabel: "TESTDISK",
		Format:      "1440",
	})
	if err != nil {
		t.Fatalf("CreateImage() error = %v", err)
	}

	dest := filepath.Join(t.TempDir(), "extract")
	err = ExtractImage(context.Background(), image, dest)
	if err != nil {
		t.Fatalf("ExtractImage() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dest, "README.TXT"))
	if err != nil {
		t.Fatalf("ReadFile(README.TXT): %v", err)
	}
	if string(data) != "hello floppy" {
		t.Fatalf("README.TXT = %q, want %q", string(data), "hello floppy")
	}

	data, err = os.ReadFile(filepath.Join(dest, "BIN", "RUN.EXE"))
	if err != nil {
		t.Fatalf("ReadFile(BIN/RUN.EXE): %v", err)
	}
	if string(data) != "binary" {
		t.Fatalf("BIN/RUN.EXE = %q, want %q", string(data), "binary")
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", path, err)
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
