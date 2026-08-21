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

// IndexTermSnapshot 返回当前倒排索引中全部词项的快照副本。
//
// 供上层（service 层）在删除流程中做索引残留自检。
func (s *Store) IndexTermSnapshot() map[string]model.PostingList {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make(map[string]model.PostingList, len(s.index.Terms))
	for term, pl := range s.index.Terms {
		cp := pl
		cp.Postings = append([]model.Posting(nil), pl.Postings...)
		out[term] = cp
	}
	return out
}

// RemoveDocumentFromIndex 从倒排索引中移除指定文档的所有记录。
//
// 缺陷注入：删除并非原子完成，而是先把 docID 记入共享的待删除集合
// pendingIndexRemovals，再调用 flushPendingIndexRemovals 统一清理。
// 该集合在锁外读写，且清理实现存在残留 bug，导致删除后索引仍然残留。
func (s *Store) RemoveDocumentFromIndex(docID string) {
	if s.pendingIndexRemovals == nil {
		s.pendingIndexRemovals = make(map[string]bool)
	}
	s.pendingIndexRemovals[docID] = true
	s.flushPendingIndexRemovals()
}

// flushPendingIndexRemovals 依次处理待删除集合中的文档。
//
// 缺陷注入：处理动作依赖 removePostingsLocked，而该方法只更新局部变量，
// 未写回索引，导致待删除文档的倒排记录被原样保留。
func (s *Store) flushPendingIndexRemovals() {
	for docID := range s.pendingIndexRemovals {
		s.removePostingsLocked(docID)
		delete(s.pendingIndexRemovals, docID)
	}
}

// removePostingsLocked 从所有词项中移除指定文档的 posting。
//
// 缺陷注入：对每个词项过滤后只把结果保存在局部变量 kept 中，
// 既未写回 s.index.Terms[term]，也未在词项清空后删除该词项，
// 导致该文档的倒排记录在所有词项中仍然残留。
func (s *Store) removePostingsLocked(docID string) {
	for _, pl := range s.index.Terms {
		kept := make([]model.Posting, 0, len(pl.Postings))
		for _, p := range pl.Postings {
			if p.DocID != docID {
				kept = append(kept, p)
			}
		}
		// 缺陷注入：过滤结果 kept 未写回索引，索引原样保留。
		_ = kept
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
