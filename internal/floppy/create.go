package floppy

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	diskfs "github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/filesystem"
)

func CreateImage(ctx context.Context, opts Options) error {
	profile, err := ParseFormat(opts.Format)
	if err != nil {
		return err
	}

	sourceDir, err := filepath.Abs(opts.SourceDir)
	if err != nil {
		return fmt.Errorf("resolve source path: %w", err)
	}

	info, err := os.Stat(sourceDir)
	if err != nil {
		return fmt.Errorf("stat source directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("source path %q is not a directory", sourceDir)
	}

	outputPath, err := filepath.Abs(opts.OutputPath)
	if err != nil {
		return fmt.Errorf("resolve output path: %w", err)
	}
	if outputPath == sourceDir || isWithin(outputPath, sourceDir) {
		return fmt.Errorf("output path %q must be outside the source directory %q", outputPath, sourceDir)
	}

	label, err := normalizeVolumeLabel(opts.VolumeLabel)
	if err != nil {
		return err
	}

	tree, summary, err := auditSource(ctx, sourceDir, label)
	if err != nil {
		return err
	}
	if summary.dataBytes > profile.DataAreaBytes {
		return fmt.Errorf("source requires %d bytes in the FAT12 data area, only %d bytes are available on a %s floppy", summary.dataBytes, profile.DataAreaBytes, profile.DisplayName)
	}
	if summary.rootEntries > profile.RootEntries {
		return fmt.Errorf("root directory needs %d entries, but FAT12 on a %s floppy only supports %d root entries", summary.rootEntries, profile.DisplayName, profile.RootEntries)
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	theDisk, err := diskfs.Create(outputPath, profile.ImageBytes, diskfs.SectorSizeDefault)
	if err != nil {
		return fmt.Errorf("create disk image: %w", err)
	}
	defer theDisk.Close()

	imgFS, err := theDisk.CreateFilesystem(disk.FilesystemSpec{
		Partition:   0,
		FSType:      filesystem.TypeFat12,
		VolumeLabel: label,
	})
	if err != nil {
		return fmt.Errorf("create FAT12 filesystem: %w", err)
	}
	defer imgFS.Close()

	if err := writeTree(ctx, imgFS, tree); err != nil {
		return err
	}

	return nil
}

func auditSource(ctx context.Context, sourceDir, volumeLabel string) (*auditNode, auditSummary, error) {
	root := &auditNode{
		name:      "",
		hostPath:  sourceDir,
		imagePath: "/",
		isDir:     true,
	}

	summary := auditSummary{}
	if volumeLabel != "" {
		summary.rootEntries = 1
	}

	if err := auditDir(ctx, root, &summary, true); err != nil {
		return nil, auditSummary{}, err
	}

	return root, summary, nil
}

func auditDir(ctx context.Context, dir *auditNode, summary *auditSummary, isRoot bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	entries, err := os.ReadDir(dir.hostPath)
	if err != nil {
		return fmt.Errorf("read directory %q: %w", dir.hostPath, err)
	}

	slices.SortFunc(entries, func(a, b fs.DirEntry) int {
		return strings.Compare(strings.ToUpper(a.Name()), strings.ToUpper(b.Name()))
	})

	seen := make(map[string]string, len(entries))
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}

		dosName, err := normalizeDOSName(entry.Name())
		if err != nil {
			return fmt.Errorf("%s: %w", filepath.Join(dir.hostPath, entry.Name()), err)
		}
		if prev, exists := seen[dosName]; exists {
			return fmt.Errorf("case-insensitive DOS name collision in %q: %q and %q both map to %q", dir.hostPath, prev, entry.Name(), dosName)
		}
		seen[dosName] = entry.Name()

		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("stat %q: %w", filepath.Join(dir.hostPath, entry.Name()), err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%q: symbolic links are not supported", filepath.Join(dir.hostPath, entry.Name()))
		}
		if !info.Mode().IsRegular() && !info.IsDir() {
			return fmt.Errorf("%q: only regular files and directories are supported", filepath.Join(dir.hostPath, entry.Name()))
		}

		child := &auditNode{
			name:      dosName,
			hostPath:  filepath.Join(dir.hostPath, entry.Name()),
			imagePath: joinImagePath(dir.imagePath, dosName),
			isDir:     info.IsDir(),
			size:      info.Size(),
		}
		dir.children = append(dir.children, child)

		if isRoot {
			summary.rootEntries++
		}

		if info.IsDir() {
			if err := auditDir(ctx, child, summary, false); err != nil {
				return err
			}
			summary.dataBytes += directoryBytes(len(child.children) + 2)
			continue
		}

		summary.dataBytes += allocatedBytes(info.Size())
	}

	return nil
}

func writeTree(ctx context.Context, imgFS filesystem.FileSystem, root *auditNode) error {
	for _, child := range root.children {
		if err := writeNode(ctx, imgFS, child); err != nil {
			return err
		}
	}
	return nil
}

func writeNode(ctx context.Context, imgFS filesystem.FileSystem, node *auditNode) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if node.isDir {
		if err := imgFS.Mkdir(node.imagePath); err != nil {
			return fmt.Errorf("create directory %q: %w", node.imagePath, err)
		}
		for _, child := range node.children {
			if err := writeNode(ctx, imgFS, child); err != nil {
				return err
			}
		}
		return nil
	}

	src, err := os.Open(node.hostPath)
	if err != nil {
		return fmt.Errorf("open source file %q: %w", node.hostPath, err)
	}
	defer src.Close()

	dst, err := imgFS.OpenFile(node.imagePath, os.O_CREATE|os.O_RDWR|os.O_TRUNC)
	if err != nil {
		return fmt.Errorf("create image file %q: %w", node.imagePath, err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("copy %q into image: %w", node.hostPath, err)
	}

	return nil
}
