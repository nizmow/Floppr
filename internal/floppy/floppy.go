package floppy

import (
	"fmt"
	"path/filepath"
	"strings"
)

const (
	floppySectorBytes int64 = 512
	reservedSectors   int64 = 1
	fatCopies         int64 = 2
)

type Options struct {
	SourceDir   string
	OutputPath  string
	VolumeLabel string
	Format      string
}

type FormatProfile struct {
	Name          string
	DisplayName   string
	TotalSectors  int64
	FATSectors    int64
	RootEntries   int
	ImageBytes    int64
	DataAreaBytes int64
}

var formatProfiles = map[string]FormatProfile{
	"360":  newFormatProfile("360", "360KB", 720, 2, 112),
	"720":  newFormatProfile("720", "720KB", 1440, 3, 112),
	"1200": newFormatProfile("1200", "1.2MB", 2400, 7, 224),
	"1440": newFormatProfile("1440", "1.44MB", 2880, 9, 224),
}

type auditNode struct {
	name      string
	hostPath  string
	imagePath string
	isDir     bool
	size      int64
	children  []*auditNode
}

type auditSummary struct {
	dataBytes   int64
	rootEntries int
}

func newFormatProfile(name, displayName string, totalSectors, fatSectors int64, rootEntries int) FormatProfile {
	rootDirSectors := int64(rootEntries*32) / floppySectorBytes
	imageBytes := totalSectors * floppySectorBytes
	dataAreaBytes := (totalSectors - reservedSectors - fatCopies*fatSectors - rootDirSectors) * floppySectorBytes
	return FormatProfile{
		Name:          name,
		DisplayName:   displayName,
		TotalSectors:  totalSectors,
		FATSectors:    fatSectors,
		RootEntries:   rootEntries,
		ImageBytes:    imageBytes,
		DataAreaBytes: dataAreaBytes,
	}
}

func DefaultFormat() string {
	return "1440"
}

func ParseFormat(name string) (FormatProfile, error) {
	if name == "" {
		name = DefaultFormat()
	}
	profile, ok := formatProfiles[name]
	if !ok {
		return FormatProfile{}, fmt.Errorf("unsupported floppy format %q, expected one of: %s", name, strings.Join(SupportedFormats(), ", "))
	}
	return profile, nil
}

func SupportedFormats() []string {
	return []string{"360", "720", "1200", "1440"}
}

func DefaultOutputPath(sourceDir string) string {
	clean := filepath.Clean(sourceDir)
	return filepath.Join(filepath.Dir(clean), sourceBaseName(clean)+".img")
}

func DefaultVolumeLabel(sourceDir string) string {
	base := dosSafeUpper(sourceBaseName(filepath.Clean(sourceDir)))
	if base == "FLOPPR" {
		return "FLOPPR"
	}
	return base
}

func sourceBaseName(path string) string {
	base := filepath.Base(path)
	if base == "." || base == string(filepath.Separator) || base == "" {
		return "floppy"
	}
	return base
}

func allocatedBytes(size int64) int64 {
	if size <= 0 {
		return 0
	}
	return ((size + floppySectorBytes - 1) / floppySectorBytes) * floppySectorBytes
}

func directoryBytes(entries int) int64 {
	if entries <= 0 {
		return floppySectorBytes
	}
	bytes := int64(entries * 32)
	return ((bytes + floppySectorBytes - 1) / floppySectorBytes) * floppySectorBytes
}

func isWithin(path, root string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}
