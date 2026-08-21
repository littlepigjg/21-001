package tokenizer

import "strings"

func Stem(word string) string {
	word = strings.ToLower(word)
	if len(word) <= 3 {
		return word
	}

	suffixes := []struct {
		suffix  string
		minStem int
	}{
		{"ization", 5},
		{"ational", 5},
		{"fulness", 5},
		{"ing", 4},
		{"edly", 4},
		{"ed", 3},
		{"ies", 3},
		{"es", 3},
		{"ly", 4},
		{"s", 3},
	}

	for _, sf := range suffixes {
		if strings.HasSuffix(word, sf.suffix) {
			stem := strings.TrimSuffix(word, sf.suffix)
			if len(stem) >= sf.minStem {
				return stem
			}
		}
	}
	return word
}

func NormalizeNumber(word string) string {
	hasDigit := false
	for _, r := range word {
		if r >= '0' && r <= '9' {
			hasDigit = true
			break
		}
	}
	if hasDigit {
		return "#num"
	}
	return word
}
