package store

import (
	"benzhi/internal/model"
	"benzhi/pkg/util"
)

// AddPosting 向倒排索引中为指定词项添加或更新一条文档记录。
//
// 该方法只更新内存索引，不触发落盘；索引构建完成后由上层调用 FlushIndex。
func (s *Store) AddPosting(term, docID string, positions []int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.addPostingLocked(term, docID, positions)
}

// addPostingLocked 在持锁状态下执行实际的新增逻辑。
func (s *Store) addPostingLocked(term, docID string, positions []int) {
	pl, ok := s.index.Terms[term]
	if !ok {
		pl = model.PostingList{Term: term, Postings: []model.Posting{}}
	}

	// 查找该文档是否已存在，存在则更新，否则追加。
	found := false
	for i := range pl.Postings {
		if pl.Postings[i].DocID == docID {
			pl.Postings[i].TermFreq = len(positions)
			pl.Postings[i].Positions = positions
			found = true
			break
		}
	}
	if !found {
		pl.Postings = append(pl.Postings, model.Posting{
			DocID:     docID,
			TermFreq:  len(positions),
			Positions: positions,
		})
	}
	pl.DocFreq = len(pl.Postings)
	s.index.Terms[term] = pl
}

// GetPostingList 返回指定词项的倒排列表，不存在时返回零值列表。
func (s *Store) GetPostingList(term string) model.PostingList {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.index.Lookup(term)
}

// NeedsIndexRebuild 报告倒排索引的词项 map 是否尚未初始化，需要重建。
//
// 加载空索引后，词项 map 可能保持 nil，此时返回 true；上层应据此决定是否
// 在首次写入前完成索引初始化。
func (s *Store) NeedsIndexRebuild() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.index == nil || s.index.Terms == nil
}

// EnsureIndexInit 保证倒排索引的词项 map 已初始化为可写状态。
//
// 适用于加载空索引、且无历史文档可触发重建的场景：迁移路径不会重建词项，
// 此时调用本方法把 nil map 初始化为空 map，避免首次写入触发 nil map panic。
// 已初始化时为空操作。
func (s *Store) EnsureIndexInit() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.index == nil {
		s.index = model.NewInvertedIndex()
		return
	}
	if s.index.Terms == nil {
		s.index.Terms = make(map[string]model.PostingList)
	}
}

// RemoveDocumentFromIndex 从倒排索引中移除指定文档的所有记录。
func (s *Store) RemoveDocumentFromIndex(docID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for term, pl := range s.index.Terms {
		out := pl.Postings[:0]
		for _, p := range pl.Postings {
			if p.DocID != docID {
				out = append(out, p)
			}
		}
		if len(out) == 0 {
			delete(s.index.Terms, term)
			continue
		}
		pl.Postings = out
		pl.DocFreq = len(out)
		s.index.Terms[term] = pl
	}
}

// IndexDocCount 返回当前索引覆盖的文档总数（由上层维护）。
func (s *Store) IndexDocCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.index.DocCount
}

// SetIndexDocCount 设置索引覆盖的文档总数。
func (s *Store) SetIndexDocCount(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.index.DocCount = n
}

// IndexTermCount 返回索引中的词项总数。
func (s *Store) IndexTermCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.index.Terms)
}

// SyncIndexDocCount 以当前文档表长度同步索引的文档计数。
func (s *Store) SyncIndexDocCount() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.index.DocCount = len(s.documents)
}

// ClearIndex 清空倒排索引（保留结构）。
func (s *Store) ClearIndex() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.index.Terms = make(map[string]model.PostingList)
	s.index.DocCount = 0
}

// FlushIndex 仅将倒排索引落盘。
func (s *Store) FlushIndex() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return util.SaveJSON(s.path(s.cfg.IndexFile), s.index)
}
