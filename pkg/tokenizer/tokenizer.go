package tokenizer

import (
	"strings"
	"unicode"
)

type Tokenizer struct {
	stopwords map[string]struct{}
}

func New() *Tokenizer {
	return &Tokenizer{stopwords: buildStopwords()}
}

func NewWithStopwords(words []string) *Tokenizer {
	set := make(map[string]struct{}, len(words))
	for _, w := range words {
		set[strings.ToLower(strings.TrimSpace(w))] = struct{}{}
	}
	return &Tokenizer{stopwords: set}
}

func (t *Tokenizer) Tokenize(text string) []string {
	if text == "" {
		return nil
	}

	text = normalize(text)

	var tokens []string
	runes := []rune(text)

	for i := 0; i < len(runes); {
		r := runes[i]
		if isCJK(r) {

			j := i
			for j < len(runes) && isCJK(runes[j]) {
				j++
			}
			tokens = append(tokens, t.tokenizeCJK(runes[i:j])...)
			i = j
		} else if unicode.IsLetter(r) || unicode.IsDigit(r) {

			j := i
			for j < len(runes) && (unicode.IsLetter(runes[j]) || unicode.IsDigit(runes[j])) {
				j++
			}
			word := strings.ToLower(string(runes[i:j]))
			if !t.isStopword(word) && len(word) >= 2 {
				tokens = append(tokens, word)
			}
			i = j
		} else {
			i++
		}
	}

	return tokens
}

func (t *Tokenizer) tokenizeCJK(runes []rune) []string {
	var out []string
	if len(runes) == 0 {
		return out
	}

	if len(runes) == 1 {
		w := string(runes)
		if !t.isStopword(w) {
			out = append(out, w)
		}
		return out
	}

	for i := 0; i+1 < len(runes); i++ {
		w := string(runes[i : i+2])
		if t.isStopword(w) {
			continue
		}
		out = append(out, w)
	}
	return out
}

func (t *Tokenizer) isStopword(w string) bool {
	if t == nil || t.stopwords == nil {
		return false
	}
	_, ok := t.stopwords[w]
	return ok
}

func isCJK(r rune) bool {
	return r >= 0x4E00 && r <= 0x9FFF
}

func tokenizeWord(segment string) []string {
	return strings.Fields(segment)
}

func (t *Tokenizer) CountTerms(text string) map[string]int {
	counts := make(map[string]int)
	for _, tok := range t.Tokenize(text) {
		counts[tok]++
	}
	return counts
}
