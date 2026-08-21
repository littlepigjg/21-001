package service

import (
	"strings"
	"unicode/utf8"

	"benzhi/internal/model"
)

const (
	HighlightStart = "[["
	HighlightEnd   = "]]"
)

func (s *Service) BuildSnippet(doc *model.Document, queryTerms []string, maxRunes int) string {
	if doc == nil || maxRunes <= 0 {
		return ""
	}

	content := []rune(doc.Content)
	if len(content) == 0 {
		return ""
	}

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
