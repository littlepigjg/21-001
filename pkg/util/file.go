package util

import (
	"os"
	"path/filepath"
	"strings"
)

// DetectFormat 根据文件名后缀推断文档格式。
//
// 返回值为内部统一格式（txt/md/pdf），未知格式返回空字符串。
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

// EnsureDir 确保目录存在，不存在则递归创建。
func EnsureDir(dir string) error {
	if dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}

// BaseWithoutExt 返回去除扩展名后的文件名。
func BaseWithoutExt(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}

// SanitizeFilename 移除文件名中的路径分隔符，避免路径穿越。
func SanitizeFilename(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	name = filepath.Base(name)
	return name
}
