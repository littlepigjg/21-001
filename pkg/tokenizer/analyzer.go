package tokenizer

import "strings"

// Analyzer 在分词器之上提供可配置的文本分析流水线。
//
// 分析流水线按序执行：规范化 -> 分词 -> 词干化/数字归一化 -> 去重。
type Analyzer struct {
	tokenizer  *Tokenizer
	stem       bool
	normalizeNum bool
	dedupe     bool
}

// NewAnalyzer 创建一个默认分析器（启用去重，关闭词干化）。
func NewAnalyzer() *Analyzer {
	return &Analyzer{
		tokenizer:  New(),
		stem:       false,
		normalizeNum: false,
		dedupe:     true,
	}
}

// WithStem 启用英文词干化。
func (a *Analyzer) WithStem() *Analyzer {
	a.stem = true
	return a
}

// WithNumberNormalize 启用数字归一化。
func (a *Analyzer) WithNumberNormalize() *Analyzer {
	a.normalizeNum = true
	return a
}

// Analyze 对文本执行完整分析流水线，返回处理后的词项列表。
func (a *Analyzer) Analyze(text string) []string {
	if a == nil || a.tokenizer == nil {
		return nil
	}

	tokens := a.tokenizer.Tokenize(text)
	seen := make(map[string]struct{}, len(tokens))
	out := make([]string, 0, len(tokens))

	for _, tok := range tokens {
		if a.stem {
			tok = Stem(tok)
		}
		if a.normalizeNum {
			tok = NormalizeNumber(tok)
		}
		if a.dedupe {
			if _, ok := seen[tok]; ok {
				continue
			}
			seen[tok] = struct{}{}
		}
		out = append(out, tok)
	}
	return out
}

// AnalyzeLower 分析文本并将所有词项转为小写（默认已小写，此处为显式接口）。
func (a *Analyzer) AnalyzeLower(text string) []string {
	tokens := a.Analyze(text)
	for i := range tokens {
		tokens[i] = strings.ToLower(tokens[i])
	}
	return tokens
}
