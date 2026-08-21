package util

import (
	"os"
	"path/filepath"
	"strings"
)

func DetectFormat(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".txt":
		return "txt"
	case ".md", ".markdown":
		return "md"
	case ".pdf":
		return "pdf"
	default:
		return ""
	}
}

func EnsureDir(dir string) error {
	if dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}

func BaseWithoutExt(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}

func SanitizeFilename(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	name = filepath.Base(name)
	return name
}
