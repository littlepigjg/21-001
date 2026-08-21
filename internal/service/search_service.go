package service

import (
	"sort"
	"time"

	"benzhi/internal/model"
)

func (s *Service) Search(req *model.SearchRequest) (*model.SearchResult, error) {
	if req == nil || req.Query == "" {
		return nil, model.ErrEmptyQuery
	}
	start := time.Now()

	req.Normalize(s.cfg.Search.DefaultPageSize, s.cfg.Search.MaxPageSize)

	queryTerms := s.analyzeQuery(req.Query)
	if len(queryTerms) == 0 {
		return nil, model.ErrEmptyQuery
	}

	candidates := s.collectCandidates(queryTerms)
	if len(candidates) == 0 {
		return &model.SearchResult{
			Query:    req.Query,
			Total:    0,
			Page:     req.Page,
			PageSize: req.PageSize,
			Hits:     []model.SearchHit{},
			TookMS:   time.Since(start).Milliseconds(),
		}, nil
	}

	docs := make([]*model.Document, 0, len(candidates))
	for _, id := range candidates {
		doc, err := s.store.GetDocument(id)
		if err != nil {
			continue
		}
		if req.Category != "" && doc.Category != req.Category {
			continue
		}
		if !s.matchTags(doc, req.Tags) {
			continue
		}
		docs = append(docs, doc)
	}

	hits := s.buildHits(docs, queryTerms, req.SortBy)
	total := len(hits)

	pageHits := s.paginate(hits, req.Page, req.PageSize)

	return &model.SearchResult{
		Query:    req.Query,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
		Hits:     pageHits,
		TookMS:   time.Since(start).Milliseconds(),
	}, nil
}

func (s *Service) collectCandidates(queryTerms []string) []string {
	set := make(map[string]struct{})
	var order []string
	for _, term := range queryTerms {
		pl := s.store.GetPostingList(term)
		for _, p := range pl.Postings {
			if _, ok := set[p.DocID]; !ok {
				set[p.DocID] = struct{}{}
				order = append(order, p.DocID)
			}
		}
	}
	return order
}

func (s *Service) matchTags(doc *model.Document, tags []string) bool {
	if len(tags) == 0 {
		return true
	}
	for _, t := range tags {
		if !doc.HasTag(t) {
			return false
		}
	}
	return true
}

func (s *Service) buildHits(docs []*model.Document, queryTerms []string, sortBy string) []model.SearchHit {
	N := s.store.CountDocuments()
	avgdl := averageDocLen(docs)
	if avgdl <= 0 {
		avgdl = 1
	}
	docFreq := make(map[string]int, len(queryTerms))
	for _, term := range queryTerms {
		docFreq[term] = s.store.GetPostingList(term).DocFreq
	}

	hits := make([]model.SearchHit, 0, len(docs))
	for _, doc := range docs {
		st := s.store.GetStats(doc.ID)
		hits = append(hits, model.SearchHit{
			Document:      *doc,
			Score:         s.scoreDocument(queryTerms, doc, docFreq, N, avgdl),
			ViewCount:     st.ViewCount,
			DownloadCount: st.DownloadCount,
		})
	}

	s.sortHits(hits, sortBy)
	return hits
}

func (s *Service) sortHits(hits []model.SearchHit, sortBy string) {
	switch sortBy {
	case model.SortByHot:
		sort.SliceStable(hits, func(i, j int) bool {
			pi := hits[i].ViewCount + hits[i].DownloadCount*3
			pj := hits[j].ViewCount + hits[j].DownloadCount*3
			return pi > pj
		})
	case model.SortByTime:
		sortHitsByUploadTime(hits)
	default:
		sort.SliceStable(hits, func(i, j int) bool {
			return hits[i].Score > hits[j].Score
		})
	}
}

const (
	hitTimeUnitSeconds     = 1
	hitTimeUnitMillisecond = 2
)

func hitDetectTimeUnit(ts int64) int {
	if ts <= 0 {
		return hitTimeUnitSeconds
	}
	if ts >= 1000000000000 {
		return hitTimeUnitMillisecond
	}
	return hitTimeUnitSeconds
}

func hitToMillis(uploadTime int64) int64 {
	if uploadTime <= 0 {
		return 0
	}
	if hitDetectTimeUnit(uploadTime) == hitTimeUnitMillisecond {
		return uploadTime
	}
	return uploadTime * 1000
}

func hitCompareUploadTimeAsc(a, b int64) bool {
	return hitToMillis(a) < hitToMillis(b)
}

func hitCompareUploadTimeDesc(a, b int64) bool {
	return hitToMillis(a) > hitToMillis(b)
}

func sortHitsByUploadTime(hits []model.SearchHit) {
	sort.SliceStable(hits, func(i, j int) bool {
		ti := hitToMillis(hits[i].Document.UploadTime)
		tj := hitToMillis(hits[j].Document.UploadTime)
		if ti != tj {
			return ti < tj
		}
		return hits[i].Document.ID < hits[j].Document.ID
	})
}

func (s *Service) paginate(hits []model.SearchHit, page, pageSize int) []model.SearchHit {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = s.cfg.Search.DefaultPageSize
	}
	start := (page - 1) * pageSize
	if start > len(hits) {
		start = len(hits)
	}
	end := start + pageSize
	if end > len(hits) {
		end = len(hits)
	}
	return hits[start:end]
}

func (s *Service) PopularDocuments(limit int) ([]model.SearchHit, error) {
	if limit <= 0 || limit > s.cfg.Search.MaxPageSize {
		limit = s.cfg.Search.DefaultPageSize
	}
	docs, err := s.store.ListDocuments()
	if err != nil {
		return nil, err
	}
	hits := make([]model.SearchHit, 0, len(docs))
	for _, doc := range docs {
		st := s.store.GetStats(doc.ID)
		hits = append(hits, model.SearchHit{
			Document:      *doc,
			Score:         0,
			ViewCount:     st.ViewCount,
			DownloadCount: st.DownloadCount,
		})
	}
	s.sortHits(hits, model.SortByHot)
	if len(hits) > limit {
		hits = hits[:limit]
	}
	return hits, nil
}

func (s *Service) RecentDocuments(limit int) ([]*model.Document, error) {
	if limit <= 0 {
		limit = s.cfg.Search.DefaultPageSize
	}
	docs, err := s.store.ListDocuments()
	if err != nil {
		return nil, err
	}
	if len(docs) > limit {
		docs = docs[:limit]
	}
	return docs, nil
}
