package floppy

import (
	"fmt"
	"path/filepath"
	"strings"
)

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
