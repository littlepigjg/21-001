package store

import "strings"

// PrefixTerms 返回倒排索引中所有以 prefix 开头的词项。
//
// 用于检索联想（autocomplete）。结果按词项字典序排序。
func (s *Store) PrefixTerms(prefix string, limit int) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 20
	}
	var out []string
	for term := range s.index.Terms {
		if strings.HasPrefix(term, prefix) {
			out = append(out, term)
		}
	}
	// 结果排序以保证输出稳定。
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}
