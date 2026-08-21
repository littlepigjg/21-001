package tokenizer

import "strings"

// Stem 对英文单词执行轻量级词干化（去常见后缀）。
//
// 说明：完整的词干化算法（如 Porter）较复杂，本实现仅做保守的后缀剥离，
// 降低同一词根不同形态带来的召回损失，同时避免过度截断。
func Stem(word string) string {
	word = strings.ToLower(word)
	if len(word) <= 3 {
		return word
	}

	suffixes := []struct {
		suffix   string
		minStem  int
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

// NormalizeNumber 将连续数字替换为占位符，用于归一化数值差异。
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
