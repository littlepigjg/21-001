// Package tokenizer 提供面向中英文混合文本的简单分词能力。
//
// 由于系统要求仅使用标准库，本分词器不依赖任何第三方分词库，采用如下策略：
//   - 连续拉丁字母/数字序列按空格与标点切分为单词；
//   - 连续中文字符按二元组（bigram）切分，兼顾召回率；
//   - 过滤停用词与过短的词项。
package tokenizer

import (
	"strings"
	"unicode"
)

// Tokenizer 是分词器结构，持有一份停用词表。
type Tokenizer struct {
	stopwords map[string]struct{}
}

// New 创建一个使用内置停用词表的分词器。
func New() *Tokenizer {
	return &Tokenizer{stopwords: buildStopwords()}
}

// NewWithStopwords 创建一个使用自定义停用词表的分词器。
func NewWithStopwords(words []string) *Tokenizer {
	set := make(map[string]struct{}, len(words))
	for _, w := range words {
		set[strings.ToLower(strings.TrimSpace(w))] = struct{}{}
	}
	return &Tokenizer{stopwords: set}
}

// Tokenize 将输入文本切分为去重前的词项序列。
//
// 返回结果保留了词项在原文中的出现顺序（包含重复），便于计算词频与位置。
func (t *Tokenizer) Tokenize(text string) []string {
	if text == "" {
		return nil
	}

	text = normalize(text)
	// 使用 rune 分段，区分 CJK 区间与拉丁区间。
	var tokens []string
	runes := []rune(text)

	for i := 0; i < len(runes); {
		r := runes[i]
		if isCJK(r) {
			// 收集连续的 CJK 字符段，然后做 bigram 切分。
			j := i
			for j < len(runes) && isCJK(runes[j]) {
				j++
			}
			tokens = append(tokens, t.tokenizeCJK(runes[i:j])...)
			i = j
		} else if unicode.IsLetter(r) || unicode.IsDigit(r) {
			// 收集连续的拉丁/数字段，然后按非字母数字边界切分。
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

// tokenizeCJK 对连续中文字符串做 bigram 切分，并过滤停用词。
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

// isStopword 判断词项是否为停用词。
func (t *Tokenizer) isStopword(w string) bool {
	if t == nil || t.stopwords == nil {
		return false
	}
	_, ok := t.stopwords[w]
	return ok
}

// isCJK 判断 rune 是否为中日韩统一表意文字区间的字符。
func isCJK(r rune) bool {
	return r >= 0x4E00 && r <= 0x9FFF
}

// tokenizeWord 将一段拉丁文本按空白切分为单词（已由 Tokenize 预处理为单词段，此函数保留兼容）。
func tokenizeWord(segment string) []string {
	return strings.Fields(segment)
}

// CountTerms 统计文本中每个词项的出现次数。
func (t *Tokenizer) CountTerms(text string) map[string]int {
	counts := make(map[string]int)
	for _, tok := range t.Tokenize(text) {
		counts[tok]++
	}
	return counts
}
