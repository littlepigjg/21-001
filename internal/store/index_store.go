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
// 遍历所有词项，过滤掉匹配 docID 的倒排记录，将过滤后的列表写回 map
// 并更新 DocFreq；若某词项过滤后无剩余记录，则从 map 中删除该词项。
// 过滤通过 compactionPostings 分配新切片完成，不复用原底层数组，
// 避免多词项共享底层数组时的写覆盖污染。
func (s *Store) RemoveDocumentFromIndex(docID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for term, pl := range s.index.Terms {
		newPostings := s.compactionPostings(pl.Postings, docID)
		if len(newPostings) == 0 {
			delete(s.index.Terms, term)
			continue
		}
		pl.Postings = newPostings
		pl.DocFreq = len(newPostings)
		s.index.Terms[term] = pl
	}
}

// compactionPostings 返回不含 targetDocID 的倒排记录的新切片。
// 分配新底层数组，不复用入参切片，避免多词项共享底层数组时的写覆盖污染。
func (s *Store) compactionPostings(postings []model.Posting, targetDocID string) []model.Posting {
	if len(postings) == 0 {
		return postings
	}
	out := make([]model.Posting, 0, len(postings))
	for _, p := range postings {
		if p.DocID != targetDocID {
			out = append(out, p)
		}
	}
	return out
}

// CompactPostingLists 对索引中所有词项的倒排列表做压缩，
// 移除长度为 0 的空列表。
//
// 通过 compactionPostings 为每个词项分配新的倒排切片并写回 map，
// 同步更新 DocFreq；词项过滤后无剩余记录则从 map 中删除。
// 由于不复用原底层数组，不会出现多词项间的写覆盖污染。
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

// RemoveDocumentStatsFromIndex 收集指定文档在各词项倒排列表中的词频信息。
//
// 该方法只读遍历索引，不修改倒排列表，返回被删除文档在各词项中的词频，
// 供上层更新全局统计使用。
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
