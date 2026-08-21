package service

import (
	"strings"
)

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
