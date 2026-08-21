package service

import (
	"strings"
)

type ParsedQuery struct {
	Terms []string

	Phrase string

	Category string

	Tags []string
}

func (s *Service) ParseQuery(raw string) ParsedQuery {
	var pq ParsedQuery
	runes := []rune(strings.TrimSpace(raw))
	i := 0

	for i < len(runes) {

		for i < len(runes) && (runes[i] == ' ' || runes[i] == '\t') {
			i++
		}
		if i >= len(runes) {
			break
		}

		if runes[i] == '"' || runes[i] == '\'' {
			quote := runes[i]
			i++
			var b strings.Builder
			for i < len(runes) && runes[i] != quote {
				b.WriteRune(runes[i])
				i++
			}
			if i < len(runes) {
				i++
			}
			if b.Len() > 0 {
				pq.Phrase = b.String()
			}
			continue
		}

		var b strings.Builder
		for i < len(runes) && runes[i] != ' ' && runes[i] != '\t' {
			b.WriteRune(runes[i])
			i++
		}
		word := b.String()

		if idx := strings.Index(word, ":"); idx > 0 {
			field := word[:idx]
			value := word[idx+1:]
			switch field {
			case "category":
				pq.Category = value
			case "tag":
				if value != "" {
					pq.Tags = append(pq.Tags, value)
				}
			default:
				pq.Terms = append(pq.Terms, word)
			}
			continue
		}

		pq.Terms = append(pq.Terms, word)
	}

	return pq
}

func (pq *ParsedQuery) CombineQuery() string {
	parts := make([]string, 0, len(pq.Terms)+1)
	parts = append(parts, pq.Terms...)
	if pq.Phrase != "" {
		parts = append(parts, pq.Phrase)
	}
	return strings.Join(parts, " ")
}
