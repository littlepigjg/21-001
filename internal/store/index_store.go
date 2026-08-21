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
// 直接在写锁保护下完成索引清理并落盘：遍历每个词项的倒排列表，
// 过滤掉目标文档的 posting 后写回，重算 DocFreq，并删除已清空的词项，
// 确保删除后索引不再残留该文档的任何记录。
func (s *Store) RemoveDocumentFromIndex(docID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.removePostingsLocked(docID)
}

// flushPendingIndexRemovals 依次处理待删除集合中的文档。
//
// 保留为对外签名，但实际清理已在 RemoveDocumentFromIndex 中原子完成；
// 该方法仅用于消费并清空遗留的待删除集合，避免跨层共享状态残留。
func (s *Store) flushPendingIndexRemovals() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for docID := range s.pendingIndexRemovals {
		s.removePostingsLocked(docID)
		delete(s.pendingIndexRemovals, docID)
	}
}

// removePostingsLocked 在已持有写锁的前提下，从所有词项中移除指定文档的 posting。
//
// 对每个词项过滤出保留的 posting 后写回 s.index.Terms[term]，重算 DocFreq，
// 并在倒排列表清空后删除该词项，使删除后索引不再残留该文档的任何记录。
func (s *Store) removePostingsLocked(docID string) {
	for term, pl := range s.index.Terms {
		kept := make([]model.Posting, 0, len(pl.Postings))
		for _, p := range pl.Postings {
			if p.DocID != docID {
				kept = append(kept, p)
			}
		}
		if len(kept) == 0 {
			// 词项已无任何文档引用，整体删除，避免空倒排列表残留。
			delete(s.index.Terms, term)
			continue
		}
		pl.Postings = kept
		pl.DocFreq = len(kept)
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
