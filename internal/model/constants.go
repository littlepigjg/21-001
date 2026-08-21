package model

// 定义检索排序方式常量。
const (
	// SortByRelevance 按相关度排序。
	SortByRelevance = "relevance"
	// SortByHot 按热度排序。
	SortByHot = "hot"
	// SortByTime 按上传时间排序。
	SortByTime = "time"
)

// 定义支持的文档格式常量。
const (
	// FormatTXT 纯文本文件。
	FormatTXT = "txt"
	// FormatMarkdown Markdown 文件。
	FormatMarkdown = "md"
	// FormatMarkdownLong Markdown 文件的长格式别名。
	FormatMarkdownLong = "markdown"
	// FormatPDF PDF 文件（仅支持简单文本抽取）。
	FormatPDF = "pdf"
)

// IsSupportedFormat 判断给定格式是否属于系统支持的范围。
func IsSupportedFormat(format string) bool {
	switch format {
	case FormatTXT, FormatMarkdown, FormatMarkdownLong, FormatPDF:
		return true
	default:
		return false
	}
}

// NormalizeFormat 将格式别名统一为内部标准格式。
func NormalizeFormat(format string) string {
	switch format {
	case FormatMarkdownLong:
		return FormatMarkdown
	case FormatTXT, FormatMarkdown, FormatPDF:
		return format
	default:
		return ""
	}
}
