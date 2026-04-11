package floppy

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	diskfs "github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/filesystem"
)

func ExtractImage(ctx context.Context, imagePath, destinationDir string) error {
	sourcePath, err := filepath.Abs(imagePath)
	if err != nil {
		return fmt.Errorf("resolve image path: %w", err)
	}

	info, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("stat image path: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("image path %q is a directory", sourcePath)
	}

	destPath, err := filepath.Abs(destinationDir)
	if err != nil {
		return fmt.Errorf("resolve destination path: %w", err)
	}
	if isWithin(destPath, sourcePath) {
		return fmt.Errorf("destination path %q must not be inside image path %q", destPath, sourcePath)
	}
	if err := os.MkdirAll(destPath, 0o755); err != nil {
		return fmt.Errorf("create destination directory: %w", err)
	}

	theDisk, err := diskfs.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("open disk image: %w", err)
	}
	defer theDisk.Close()

	imgFS, err := theDisk.GetFilesystem(0)
	if err != nil {
		return fmt.Errorf("open filesystem: %w", err)
	}
	defer imgFS.Close()

	if err := extractDir(ctx, imgFS, ".", destPath); err != nil {
		return err
	}

	return nil
}

func extractDir(ctx context.Context, imgFS filesystem.FileSystem, imageDir, hostDir string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	entries, err := imgFS.ReadDir(imageDir)
	if err != nil {
		return fmt.Errorf("read image directory %q: %w", imageDir, err)
	}

	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}

		imagePath := path.Join(imageDir, entry.Name())
		hostPath := filepath.Join(hostDir, entry.Name())
		if entry.IsDir() {
			if err := os.MkdirAll(hostPath, 0o755); err != nil {
				return fmt.Errorf("create host directory %q: %w", hostPath, err)
			}
			if err := extractDir(ctx, imgFS, imagePath, hostPath); err != nil {
				return err
			}
			continue
		}

		if err := extractFile(ctx, imgFS, imagePath, hostPath); err != nil {
			return err
		}
	}

	return nil
}

func extractFile(ctx context.Context, imgFS filesystem.FileSystem, imagePath, hostPath string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	src, err := imgFS.Open(imagePath)
	if err != nil {
		return fmt.Errorf("open image file %q: %w", imagePath, err)
	}
	defer src.Close()

	dst, err := os.OpenFile(hostPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("create host file %q: %w", hostPath, err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("copy image file %q to host path %q: %w", imagePath, hostPath, err)
	}

	return nil
}
