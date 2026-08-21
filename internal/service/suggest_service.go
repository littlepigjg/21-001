package service

import (
	"strings"
)

// Suggest 根据前缀返回检索联想词项。
//
// prefix 为空时返回空列表。该功能基于倒排索引中的词项做前缀匹配。
func (s *Service) Suggest(prefix string, limit int) []string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return nil
	}
	if limit <= 0 {
		limit = 10
	}
	return s.store.PrefixTerms(strings.ToLower(prefix), limit)
}
