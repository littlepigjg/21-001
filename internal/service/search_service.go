package service

import (
	"sort"
	"time"

	"benzhi/internal/model"
)

// Search 执行全文检索，返回按指定方式排序并分页后的结果。
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

	// 收集候选文档 ID（取各查询词倒排列表的并集）。
	candidates := s.collectCandidates(queryTerms)

	// 短语检索：要求词项按顺序连续出现（基于倒排位置做邻近匹配）。
	if req.Phrase != "" {
		candidates = s.filterPhrase(candidates, s.analyzeQuery(req.Phrase))
	}

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

	// 加载候选文档并应用分类/标签过滤。
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

	// 构造命中项并排序。
	hits := s.buildHits(docs, queryTerms, req.SortBy)
	total := len(hits)

	// 分页。
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

// collectCandidates 返回所有查询词倒排列表中文档 ID 的并集。
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

// filterPhrase 仅保留查询词项按顺序连续出现的文档。
func (s *Service) filterPhrase(docIDs []string, terms []string) []string {
	if len(terms) < 2 {
		return docIDs
	}
	out := make([]string, 0, len(docIDs))
	for _, id := range docIDs {
		if s.phraseMatches(id, terms) {
			out = append(out, id)
		}
	}
	return out
}

// phraseMatches 判断文档中 terms 是否按顺序连续出现。
func (s *Service) phraseMatches(docID string, terms []string) bool {
	posLists := make([][]int, len(terms))
	for i, term := range terms {
		pl := s.store.GetPostingList(term)
		found := false
		for _, p := range pl.Postings {
			if p.DocID == docID {
				posLists[i] = p.Positions
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	for _, start := range posLists[0] {
		if matchPhraseChain(posLists, start) {
			return true
		}
	}
	return false
}

// matchPhraseChain 从 start 开始判断后续词项是否位置连续递增。
func matchPhraseChain(posLists [][]int, start int) bool {
	cur := start
	for j := 1; j < len(posLists); j++ {
		cur++
		if !containsPosition(posLists[j], cur) {
			return false
		}
	}
	return true
}

// containsPosition 判断位置切片中是否包含指定位置。
func containsPosition(list []int, pos int) bool {
	for _, v := range list {
		if v == pos {
			return true
		}
	}
	return false
}

// matchTags 判断文档是否满足标签过滤条件（要求包含全部指定标签）。
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

// buildHits 根据文档构造命中项，并按排序方式排序。
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

// sortHits 按指定方式排序命中列表。
func (s *Service) sortHits(hits []model.SearchHit, sortBy string) {
	switch sortBy {
	case model.SortByHot:
		sort.SliceStable(hits, func(i, j int) bool {
			pi := hits[i].ViewCount + hits[i].DownloadCount*3
			pj := hits[j].ViewCount + hits[j].DownloadCount*3
			return pi > pj
		})
	case model.SortByTime:
		sort.SliceStable(hits, func(i, j int) bool {
			return hits[i].Document.UploadTime > hits[j].Document.UploadTime
		})
	default:
		sort.SliceStable(hits, func(i, j int) bool {
			return hits[i].Score > hits[j].Score
		})
	}
}

// paginate 对命中列表做分页切片。
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

// PopularDocuments 返回按热度排序的文档列表。
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

// RecentDocuments 返回最近上传的文档列表。
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
