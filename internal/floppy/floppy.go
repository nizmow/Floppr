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

func normalizeVolumeLabel(label string) (string, error) {
	label = strings.TrimSpace(strings.ToUpper(label))
	if label == "" {
		return "", nil
	}
	if len(label) > 11 {
		return "", fmt.Errorf("volume label %q exceeds DOS 11-character limit", label)
	}
	if strings.Contains(label, ".") {
		return "", fmt.Errorf("volume label %q must not contain dots", label)
	}
	if err := validateDOSChars(label, fmt.Sprintf("volume label %q", label)); err != nil {
		return "", err
	}
	return label, nil
}

func normalizeDOSName(name string) (string, error) {
	if name == "" || name == "." || name == ".." {
		return "", fmt.Errorf("invalid DOS filename %q", name)
	}

	upper := strings.ToUpper(name)
	parts := strings.Split(upper, ".")
	if len(parts) > 2 {
		return "", fmt.Errorf("name %q is not DOS 8.3 compatible", name)
	}

	base := parts[0]
	if err := validateDOSPart(base, 8, fmt.Sprintf("base name %q", name)); err != nil {
		return "", err
	}

	if len(parts) == 1 {
		return base, nil
	}

	ext := parts[1]
	if err := validateDOSPart(ext, 3, fmt.Sprintf("extension %q", name)); err != nil {
		return "", err
	}

	return base + "." + ext, nil
}

func joinImagePath(parent, name string) string {
	path := filepath.ToSlash(filepath.Join(parent, name))
	if strings.HasPrefix(path, "/") {
		return path
	}
	return "/" + path
}

func dosSafeUpper(name string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(name) {
		if b.Len() == 11 {
			break
		}
		if r == '.' || r == ' ' {
			continue
		}
		if isDOSChar(r) {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "FLOPPR"
	}
	return b.String()
}

func validateDOSPart(value string, maxLen int, what string) error {
	if value == "" || len(value) > maxLen {
		return fmt.Errorf("%s must be 1-%d DOS characters", what, maxLen)
	}
	return validateDOSChars(value, what)
}

func validateDOSChars(value, what string) error {
	for _, r := range value {
		if !isDOSChar(r) {
			return fmt.Errorf("%s contains unsupported DOS character %q", what, r)
		}
	}
	return nil
}

func isDOSChar(r rune) bool {
	switch {
	case r >= 'A' && r <= 'Z':
		return true
	case r >= '0' && r <= '9':
		return true
	}

	switch r {
	case '!', '#', '$', '%', '&', '\'', '(', ')', '-', '@', '^', '_', '`', '{', '}', '~':
		return true
	default:
		return false
	}
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
