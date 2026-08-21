package tokenizer

import (
	"strings"
	"unicode"
)

// normalize 将文本统一为小写，并把标点、空白替换为空格。
//
// 保留中文字符与字母数字，其余字符统一替换为空格以分隔词项。
func normalize(text string) string {
	var b strings.Builder
	b.Grow(len(text))

	for _, r := range text {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r):
			b.WriteRune(unicode.ToLower(r))
		case unicode.IsSpace(r):
			b.WriteRune(' ')
		default:
			// 其他字符（标点、符号）替换为空格，避免粘连。
			b.WriteRune(' ')
		}
	}

	return b.String()
}

// buildStopwords 构建内置的停用词表。
//
// 停用词覆盖常见中文虚词与英文冠词/介词，降低索引噪音。
func buildStopwords() map[string]struct{} {
	words := []string{
		// 中文单字与常见虚词
		"的", "了", "和", "是", "在", "我", "有", "就", "不", "人", "都",
		"一", "个", "上", "也", "很", "到", "说", "要", "去", "你", "会",
		"着", "没", "看", "好", "这", "那", "他", "她", "它", "们", "与",
		"及", "或", "为", "被", "把", "让", "对", "从", "向", "于", "等",
		"而", "且", "但", "却", "则", "其", "之", "如", "若", "因", "所",
		"并", "无", "非", "由", "以", "可", "能", "还", "又", "再", "才",
		"已", "将", "正", "这", "些", "些", "什么", "怎么", "如何", "为什么",
		// 英文常见停用词
		"the", "a", "an", "and", "or", "but", "if", "then", "else", "of",
		"to", "in", "on", "at", "for", "with", "by", "from", "as", "is",
		"are", "was", "were", "be", "been", "being", "it", "its", "this",
		"that", "these", "those", "you", "your", "he", "she", "they", "we",
		"not", "no", "do", "does", "did", "have", "has", "had", "will",
		"would", "can", "could", "should", "may", "might", "must", "i",
		"am", "about", "above", "after", "again", "all", "any", "because",
	}
	set := make(map[string]struct{}, len(words))
	for _, w := range words {
		set[w] = struct{}{}
	}
	return set
}
