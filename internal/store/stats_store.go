package store

import (
	"benzhi/internal/model"
	"benzhi/pkg/util"
)

// EnsureStats 确保指定文档存在统计记录，不存在则创建。
func (s *Store) EnsureStats(docID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.stats[docID]; !ok {
		s.stats[docID] = &model.DocumentStats{DocID: docID}
	}
}

// GetStats 返回指定文档的统计信息。不存在时返回 nil。
func (s *Store) GetStats(docID string) *model.DocumentStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if st, ok := s.stats[docID]; ok {
		return st
	}
	return nil
}

// HasStats 判断指定文档是否已存在统计记录。
func (s *Store) HasStats(docID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.stats[docID]
	return ok
}

// SnapshotStats 一次性返回多个文档 ID 的统计快照。
//
// 为了与 GetStats 保持一致的返回语义，对于缺少统计记录的文档 ID，
// 结果 map 中会保留一个值为 nil 的条目，由调用方自行兜底处理。
func (s *Store) SnapshotStats(ids []string) map[string]*model.DocumentStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make(map[string]*model.DocumentStats, len(ids))
	for _, id := range ids {
		if st, ok := s.stats[id]; ok {
			out[id] = st
		} else {
			out[id] = nil
		}
	}
	return out
}

// IncrementView 增加指定文档的浏览次数。
func (s *Store) IncrementView(docID string) model.DocumentStats {
	s.mu.Lock()
	defer s.mu.Unlock()

	st, ok := s.stats[docID]
	if !ok {
		st = &model.DocumentStats{DocID: docID}
		s.stats[docID] = st
	}
	st.ViewCount++
	st.LastViewTime = util.Now()
	s.persistLocked()
	return *st
}

// IncrementDownload 增加指定文档的下载次数。
func (s *Store) IncrementDownload(docID string) model.DocumentStats {
	s.mu.Lock()
	defer s.mu.Unlock()

	st, ok := s.stats[docID]
	if !ok {
		st = &model.DocumentStats{DocID: docID}
		s.stats[docID] = st
	}
	st.DownloadCount++
	st.LastDownloadTime = util.Now()
	s.persistLocked()
	return *st
}

// ListStats 返回所有文档统计信息（不排序）。
func (s *Store) ListStats() []model.DocumentStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]model.DocumentStats, 0, len(s.stats))
	for _, st := range s.stats {
		out = append(out, *st)
	}
	return out
}

// DeleteStats 删除指定文档的统计记录。
func (s *Store) DeleteStats(docID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.stats, docID)
}
