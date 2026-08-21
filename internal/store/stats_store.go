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

// GetStats 返回指定文档的统计信息。不存在时返回零值统计。
func (s *Store) GetStats(docID string) model.DocumentStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if st, ok := s.stats[docID]; ok {
		return *st
	}
	return model.DocumentStats{DocID: docID}
}

// IncrementView 原子地将指定文档的浏览次数加一。
//
// 读当前值、加一、写回三步全部在同一把写锁内完成，构成一个原子的
// 读-改-写，因此并发浏览之间不会相互覆盖，每次浏览都会被准确累计。
func (s *Store) IncrementView(docID string) model.DocumentStats {
	s.mu.Lock()
	defer s.mu.Unlock()

	st, ok := s.stats[docID]
	if !ok {
		st = &model.DocumentStats{DocID: docID}
		s.stats[docID] = st
	}
	st.ViewCount++
	st.Revision++
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
