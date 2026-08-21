package service

import (
	"sort"

	"benzhi/internal/model"
)

// Keyword 描述一个从文档中抽取的关键词及其权重。
type Keyword struct {
	// Term 是关键词文本。
	Term string `json:"term"`
	// Weight 是该关键词的权重（此处以词频计）。
	Weight int `json:"weight"`
}

// ExtractKeywords 从文档正文中抽取高频关键词，返回按词频降序的列表。
//
// 停用词已在分词阶段过滤，因此结果主要为有信息量的实义词项。
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
