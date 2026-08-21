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

// Terms 返回倒排索引词项映射的快照副本，供上层安全地遍历。
//
// 返回的是在持有读锁期间拷贝出的副本，调用方在锁外遍历也不会与并发写发生
// "concurrent map read and map write"。直接暴露内部 map 引用会让上层在无锁
// 状态下遍历，与并发建索引（MergePostingList）形成数据竞争，因此这里改为返回快照。
func (s *Store) Terms() map[string]model.PostingList {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make(map[string]model.PostingList, len(s.index.Terms))
	for term, pl := range s.index.Terms {
		out[term] = pl
	}
	return out
}

// CollectCandidateDocIDs 返回给定查询词项倒排列表中文档 ID 的并集。
//
// 遍历在持有读锁期间完成，避免并发建索引写 map 时触发
// "concurrent map read and map write"。结果保持首次出现顺序（去重）。
func (s *Store) CollectCandidateDocIDs(queryTerms []string) []string {
	if len(queryTerms) == 0 {
		return nil
	}

	want := make(map[string]struct{}, len(queryTerms))
	for _, t := range queryTerms {
		want[t] = struct{}{}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	set := make(map[string]struct{})
	var order []string
	for term, pl := range s.index.Terms {
		if _, ok := want[term]; !ok {
			continue
		}
		for _, p := range pl.Postings {
			if _, ok := set[p.DocID]; !ok {
				set[p.DocID] = struct{}{}
				order = append(order, p.DocID)
			}
		}
	}
	return order
}

// MergePostingList 将一个词项的倒排列表合并进内存索引。
//
// 在持有写锁的状态下读写共享的倒排索引 map，与并发检索路径（读 map）互斥，
// 避免触发 "concurrent map read and map write"。
// 若该词项此前不存在于索引中，返回 true（表示新增词项）。
func (s *Store) MergePostingList(term string, incoming model.PostingList) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.index.Terms == nil {
		s.index.Terms = make(map[string]model.PostingList)
	}

	cur, existed := s.index.Terms[term]
	if !existed {
		cur = model.PostingList{Term: term, Postings: []model.Posting{}}
	}

	// 按文档 ID 合并：已存在的文档更新词频与位置，新文档追加。
	for _, p := range incoming.Postings {
		merged := false
		for i := range cur.Postings {
			if cur.Postings[i].DocID == p.DocID {
				cur.Postings[i].TermFreq = p.TermFreq
				cur.Postings[i].Positions = p.Positions
				merged = true
				break
			}
		}
		if !merged {
			cur.Postings = append(cur.Postings, p)
		}
	}

	cur.DocFreq = len(cur.Postings)
	s.index.Terms[term] = cur
	return !existed
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
//
// 返回值是 Postings 的独立副本，调用方可在锁外安全遍历，无需担心并发写索引
// （MergePostingList 追加 postings）改写底层切片导致的数据竞争。
func (s *Store) GetPostingList(term string) model.PostingList {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pl := s.index.Lookup(term)
	if len(pl.Postings) > 0 {
		postings := append([]model.Posting(nil), pl.Postings...)
		pl.Postings = postings
	}
	return pl
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
