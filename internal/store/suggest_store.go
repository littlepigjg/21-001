package store

import "strings"

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
