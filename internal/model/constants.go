package model

const (
	SortByRelevance = "relevance"

	SortByHot = "hot"

	SortByTime = "time"
)

const (
	FormatTXT = "txt"

	FormatMarkdown = "md"

	FormatMarkdownLong = "markdown"

	FormatPDF = "pdf"
)

func IsSupportedFormat(format string) bool {
	switch format {
	case FormatTXT, FormatMarkdown, FormatMarkdownLong, FormatPDF:
		return true
	default:
		return false
	}
}

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
