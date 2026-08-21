package tokenizer

import (
	"sort"
	"unicode/utf8"
)

func Terms(text string) []string {
	t := New()
	seen := make(map[string]struct{})
	for _, tok := range t.Tokenize(text) {
		if utf8.RuneCountInString(tok) < 1 {
			continue
		}
		seen[tok] = struct{}{}
	}

	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func CountTerms(text string) map[string]int {
	t := New()
	counts := make(map[string]int)
	for _, tok := range t.Tokenize(text) {
		counts[tok]++
	}
	return counts
}
