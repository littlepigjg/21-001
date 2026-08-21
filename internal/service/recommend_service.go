package service

import (
	"sort"

	"benzhi/internal/model"
	"benzhi/pkg/util"
)

// RecommendRelated 返回与指定文档相关的文档列表（基于共享标签）。
//
// 相关度以两篇文档共享的标签数量衡量，共享标签越多排名越靠前。
func (s *Service) RecommendRelated(docID string, limit int) ([]model.SearchHit, error) {
	if limit <= 0 || limit > s.cfg.Search.MaxPageSize {
		limit = s.cfg.Search.DefaultPageSize
	}

	src, err := s.store.GetDocument(docID)
	if err != nil {
		return nil, err
	}

	all, err := s.store.ListDocuments()
	if err != nil {
		return nil, err
	}

	type scored struct {
		doc   *model.Document
		score int
	}
	var candidates []scored
	for _, d := range all {
		if d.ID == docID {
			continue
		}
		shared := sharedTagCount(src, d)
		if shared == 0 {
			continue
		}
		candidates = append(candidates, scored{doc: d, score: shared})
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		return candidates[i].doc.UploadTime > candidates[j].doc.UploadTime
	})

	hits := make([]model.SearchHit, 0, limit)
	for i, c := range candidates {
		if i >= limit {
			break
		}
		st := s.store.GetStats(c.doc.ID)
		hits = append(hits, model.SearchHit{
			Document:      *c.doc,
			Score:         float64(c.score),
			ViewCount:     st.ViewCount,
			DownloadCount: st.DownloadCount,
		})
	}
	return hits, nil
}

// sharedTagCount 计算两篇文档共享的标签数量。
func sharedTagCount(a, b *model.Document) int {
	if a == nil || b == nil {
		return 0
	}
	set := util.NewStringSetFrom(a.Tags)
	return set.Intersect(util.NewStringSetFrom(b.Tags)).Size()
}
