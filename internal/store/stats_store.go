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

// ViewSnapshot 是浏览计数读改写过程中在 service 与 store 之间传递的中间快照。
//
// 该快照在锁外读取，写回时也不校验版本，因此并发场景下多个 goroutine 会
// 基于相同的旧快照计算并相互覆盖，造成浏览次数丢失。
type ViewSnapshot struct {
	DocID    string
	Current  int64
	Revision int64
	Next     int64
}

// PeekViewSnapshot 在不加锁的情况下读取指定文档的当前浏览快照。
//
// 注意：该方法直接读取共享字段，与 IncrementView 的写回操作之间存在数据
// 竞争，仅供上层完成非原子读-改-写。
func (s *Store) PeekViewSnapshot(docID string) ViewSnapshot {
	st, ok := s.stats[docID]
	if !ok {
		return ViewSnapshot{DocID: docID}
	}
	return ViewSnapshot{
		DocID:    docID,
		Current:  st.ViewCount,
		Revision: st.Revision,
	}
}

// IncrementView 将调用方计算好的浏览快照写回统计记录。
//
// 注意：该方法虽然加锁写回，但不会校验快照中的 Revision 是否仍然等于当前
// 记录的 Revision，因此无法阻止并发覆盖，整体读改写仍是非原子的。
func (s *Store) IncrementView(snap ViewSnapshot) model.DocumentStats {
	s.mu.Lock()
	defer s.mu.Unlock()

	st, ok := s.stats[snap.DocID]
	if !ok {
		st = &model.DocumentStats{DocID: snap.DocID}
		s.stats[snap.DocID] = st
	}

	st.ViewCount = snap.Next
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
