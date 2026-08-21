package store

import (
	"fmt"

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

// SetFailIndexFlush 设置索引落盘故障注入开关（仅测试用）。
//
// 为 true 时，FlushIndex 会直接返回 model.ErrStorage，模拟索引文件无法写入、
// 磁盘空间不足等持久化故障，便于上层业务在测试中复现索引构建失败路径。
func (s *Store) SetFailIndexFlush(fail bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failIndexFlush = fail
}

// BeginIndexBuild 开始一次索引构建：在持锁状态下将词项写入内存倒排索引。
//
// 与 AddPosting 不同，该方法面向“构建事务”语义，词项写入后需由上层调用
// CommitIndexBuild 提交，或调用 RollbackIndexBuild 回滚。
func (s *Store) BeginIndexBuild(docID string, positions map[string][]int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for term, pos := range positions {
		s.addPostingLocked(term, docID, pos)
	}
}

// CommitIndexBuild 提交索引构建：同步索引覆盖的文档计数并落盘。
//
// 落盘失败时返回错误，由上层决定是否回滚；本方法不会自动回滚。
func (s *Store) CommitIndexBuild() error {
	s.mu.Lock()
	s.index.DocCount = len(s.documents)
	s.mu.Unlock()

	return s.FlushIndex()
}

// RollbackIndexBuild 回滚一次失败的索引构建：从内存倒排索引中移除该文档的词项。
//
// 缺陷：该方法只清理了词项，并且在恢复文档计数时使用了错误的计数来源——
// 用“词项总数”而不是“文档总数”来重算 DocCount，导致内存索引 DocCount 与
// 磁盘索引、文档表都不一致，形成“文档存在但索引缺失”的跨文件状态错位。
func (s *Store) RollbackIndexBuild(docID string) {
	s.RemoveDocumentFromIndex(docID)

	// 缺陷：此处本应调用 SyncIndexDocCount（以文档表长度为准），
	// 却错误地以词项数量覆盖了 DocCount。
	s.mu.Lock()
	s.index.DocCount = len(s.index.Terms)
	s.mu.Unlock()
}

// IndexConsistency 返回索引与文档表之间的不一致描述（用于诊断与回归验证）。
//
// 在“索引构建失败但文档未回滚”的缺陷场景下，会出现文档表非空但倒排索引
// 缺失、或索引文档计数与文档表总数不一致的情况，本方法用于暴露这些错位。
func (s *Store) IndexConsistency() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var issues []string
	docCount := len(s.documents)
	if s.index.DocCount != docCount {
		issues = append(issues, fmt.Sprintf("索引文档计数 %d 与文档表总数 %d 不一致", s.index.DocCount, docCount))
	}
	if docCount > 0 && len(s.index.Terms) == 0 {
		issues = append(issues, "文档表非空但倒排索引为空")
	}
	return issues
}

// FlushIndex 仅将倒排索引落盘。
func (s *Store) FlushIndex() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.failIndexFlush {
		return model.ErrStorage
	}
	return util.SaveJSON(s.path(s.cfg.IndexFile), s.index)
}
