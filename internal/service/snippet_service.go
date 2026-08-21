package service

import (
	"strings"
	"unicode/utf8"

	"benzhi/internal/model"
)

// HighlightStart 与 HighlightEnd 是命中词项的高亮标记。
const (
	HighlightStart = "[["
	HighlightEnd   = "]]"
)

// BuildSnippet 为文档正文生成包含命中词项的摘要片段。
//
// 片段优先选取第一个命中词项附近的上下文，长度不超过 maxRunes（字符数）。
// 命中词项会被 HighlightStart/HighlightEnd 包裹。
func (s *Service) BuildSnippet(doc *model.Document, queryTerms []string, maxRunes int) string {
	if doc == nil || maxRunes <= 0 {
		return ""
	}

	content := []rune(doc.Content)
	if len(content) == 0 {
		return ""
	}

	// 查找第一个命中词项在正文中的位置。
	lowerContent := strings.ToLower(doc.Content)
	firstHit := -1
	for _, term := range queryTerms {
		idx := strings.Index(lowerContent, strings.ToLower(term))
		if idx >= 0 && (firstHit == -1 || idx < firstHit) {
			firstHit = idx
		}
	}

	start := 0
	if firstHit >= 0 {
		hitRune := utf8.RuneCountInString(doc.Content[:firstHit])
		start = hitRune - maxRunes/2
		if start < 0 {
			start = 0
		}
	}

	end := start + maxRunes
	if end > len(content) {
		end = len(content)
	}

	snippet := string(content[start:end])
	// 对片段内的命中词项做高亮替换（忽略大小写）。
	lowerSnippet := strings.ToLower(snippet)
	for _, term := range queryTerms {
		lt := strings.ToLower(term)
		offset := 0
		for {
			idx := strings.Index(lowerSnippet[offset:], lt)
			if idx < 0 {
				break
			}
			abs := offset + idx
			snippet = snippet[:abs] + HighlightStart + snippet[abs:abs+len(term)] + HighlightEnd + snippet[abs+len(term):]
			// 重新计算 lowerSnippet，避免高亮标记影响后续匹配。
			lowerSnippet = strings.ToLower(snippet)
			offset = abs + len(HighlightStart) + len(term) + len(HighlightEnd)
		}
	}

	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(content) {
		snippet = snippet + "..."
	}
	return snippet
}
