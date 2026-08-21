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
// 该方法遍历所有词项的倒排列表，过滤掉匹配 docID 的记录。
// 过滤后的列表通过 compactionPostings 就地压缩，但由于使用 [:0]
// 子切片复用底层数组，且压缩结果未写回 map，导致 map 中仍保留
// 包含已删除文档记录的旧切片。
func (s *Store) RemoveDocumentFromIndex(docID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for term, pl := range s.index.Terms {
		newPostings := s.compactionPostings(pl.Postings, docID)
		if len(newPostings) == 0 {
			delete(s.index.Terms, term)
			continue
		}
		// BUG: 过滤后的 newPostings 未写回 map，map 中仍保留旧的 pl.Postings，
		// 包含已删除文档的记录。同时 compactionPostings 使用 [:0] 子切片
		// 复用底层数组，若多个词项共享同一底层数组，append 写入会覆盖
		// 其他词项的数据，造成倒排列表被污染。
		_ = pl
		_ = newPostings
	}
}

// compactionPostings 将不包含 targetDocID 的记录就地压缩到切片头部，
// 返回压缩后的子切片。
//
// 该方法使用 [:0] 创建一个复用原切片底层数组的空切片，然后通过 append
// 将不需要删除的记录写入。这种做法虽然避免了内存分配，但如果原切片与其他
// 词项的切片共享同一底层数组，append 写入会覆盖底层数组中其他词项的数据。
func (s *Store) compactionPostings(postings []model.Posting, targetDocID string) []model.Posting {
	out := postings[:0]
	for _, p := range postings {
		if p.DocID != targetDocID {
			out = append(out, p)
		}
	}
	return out
}

// CompactPostingLists 对索引中所有词项的倒排列表做就地压缩，
// 移除空的倒排列表条目。
//
// 该方法在 RemoveDocumentFromIndex 之后调用，用于清理被删除文档
// 后留下的空倒排列表。但由于使用 [:0] 子切片复用底层数组，
// 如果多个词项的 Posting 切片共享同一底层数组，压缩操作会互相
// 覆盖对方的数据。
func (s *Store) CompactPostingLists() {
	s.mu.Lock()
	defer s.mu.Unlock()

	emptyTerms := make([]string, 0)
	for term, pl := range s.index.Terms {
		compacted := s.compactionPostings(pl.Postings, "")
		if len(compacted) == 0 {
			emptyTerms = append(emptyTerms, term)
			continue
		}
		pl.Postings = compacted
		pl.DocFreq = len(compacted)
		s.index.Terms[term] = pl
	}
	for _, term := range emptyTerms {
		delete(s.index.Terms, term)
	}
}

// RemoveDocumentStatsFromIndex 清理指定文档在索引统计中的残留记录。
//
// 该方法遍历所有词项的倒排列表，统计被删除文档的词频信息，
// 用于更新索引的全局统计。由于使用了 compactionPostings 就地压缩，
// 如果多个词项共享底层数组，统计过程中可能读取到被污染的数据。
func (s *Store) RemoveDocumentStatsFromIndex(docID string) map[string]int {
	s.mu.Lock()
	defer s.mu.Unlock()

	termFreqs := make(map[string]int)
	for term, pl := range s.index.Terms {
		for _, p := range pl.Postings {
			if p.DocID == docID {
				termFreqs[term] = p.TermFreq
			}
		}
	}
	return termFreqs
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
