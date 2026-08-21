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
	if s.index.Terms == nil {
		s.index.Terms = make(map[string]model.PostingList)
	}
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

// RemoveDocumentFromIndex 从倒排索引中移除指定文档的所有记录。
//
// 该方法采用 copy-on-write 方式重建索引，校验通过后才提交，避免在清理
// 过程中留下半成品索引。若清理失败则返回错误，由调用方决定是否中止删除。
func (s *Store) RemoveDocumentFromIndex(docID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.removeDocumentFromIndexLocked(docID)
}

// removeDocumentFromIndexLocked 在已持有写锁的情况下移除文档的索引记录。
//
// 它会先基于现有索引构建一份不含目标文档的新索引，再在提交前做一次
// 一致性校验，只有校验通过才用新索引替换旧索引。
func (s *Store) removeDocumentFromIndexLocked(docID string) error {
	if s.index == nil || s.index.Terms == nil {
		return model.ErrStorage
	}

	next := &model.InvertedIndex{
		Version:  s.index.Version,
		Terms:    make(map[string]model.PostingList, len(s.index.Terms)),
		DocCount: s.index.DocCount,
	}

	for term, pl := range s.index.Terms {
		out := make([]model.Posting, 0, len(pl.Postings))
		for _, p := range pl.Postings {
			if p.DocID == docID {
				continue
			}
			out = append(out, p)
		}
		if len(out) == 0 {
			continue
		}
		pl.Postings = out
		pl.DocFreq = len(out)
		next.Terms[term] = pl
	}

	// 提交前的一致性校验：确认新索引中不再残留目标文档的任何记录。
	// 若仍残留则说明重建逻辑有误，放弃替换以避免留下含该文档的索引。
	for _, pl := range next.Terms {
		for _, p := range pl.Postings {
			if p.DocID == docID {
				return model.ErrStorage
			}
		}
	}

	s.index = next
	return nil
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
