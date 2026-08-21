package service

import (
	"sort"

	"benzhi/internal/model"
)

type Keyword struct {
	Term string `json:"term"`

	Weight int `json:"weight"`
}

func (s *Service) ExtractKeywords(doc *model.Document, limit int) []Keyword {
	if doc == nil {
		return nil
	}
	if limit <= 0 {
		limit = 10
	}

	counts := s.tokenizer.CountTerms(doc.Content)
	kws := make([]Keyword, 0, len(counts))
	for term, weight := range counts {
		kws = append(kws, Keyword{Term: term, Weight: weight})
	}

	sort.Slice(kws, func(i, j int) bool {
		if kws[i].Weight != kws[j].Weight {
			return kws[i].Weight > kws[j].Weight
		}
		return kws[i].Term < kws[j].Term
	})

	if len(kws) > limit {
		kws = kws[:limit]
	}
	return kws
}
