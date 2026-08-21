package tokenizer

import (
	"sort"
	"unicode/utf8"
)

// Terms 返回输入文本经分词与去重后的词项列表（按字典序排序）。
//
// 该函数是一个便捷入口，供需要唯一词项集合的场景使用（例如查询词分析）。
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

// CountTerms 统计文本中每个词项的出现次数。
func CountTerms(text string) map[string]int {
	t := New()
	counts := make(map[string]int)
	for _, tok := range t.Tokenize(text) {
		counts[tok]++
	}
	return counts
}
