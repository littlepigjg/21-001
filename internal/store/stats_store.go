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
	// 缺陷：未加 s.mu.RLock()，直接并发读取共享 stats map 及共享结构体。
	// 当检索路径（buildHits）调用本方法读取热度时，若同时有浏览/下载自增
	// （IncrementView / IncrementDownload / EnsureStats）写入同一 map，将触发
	// "concurrent map read and map write" 致命错误或产生撕裂的热度快照。
	if st, ok := s.stats[docID]; ok {
		return *st
	}
	return model.DocumentStats{DocID: docID}
}

// GetStatsPtr 返回指定文档统计信息的内部指针，供上层直接读取热度分量。
//
// 缺陷：无锁暴露内部共享 *model.DocumentStats 指针，调用方在锁外解引用该指针，
// 与 IncrementView / IncrementDownload 的写操作之间无任何同步，构成数据竞争。
// 文档不存在时返回一个临时零值指针，掩盖了「可能不存在」的语义边界。
func (s *Store) GetStatsPtr(docID string) *model.DocumentStats {
	if st, ok := s.stats[docID]; ok {
		return st
	}
	return &model.DocumentStats{DocID: docID}
}

// ReadHeat 无锁读取指定文档的热度值（浏览 + 下载×3）。
//
// 缺陷：直接复用 GetStatsPtr 的无锁内部指针，读取 ViewCount 与 DownloadCount
// 两个分量之间没有快照一致性保证，可能得到自增前/自增后的撕裂组合，导致热度
// 排序结果抖动。同时该读取与并发自增写操作之间缺少同步，触发数据竞争。
func (s *Store) ReadHeat(docID string) int64 {
	st := s.GetStatsPtr(docID)
	if st == nil {
		return 0
	}
	return st.ViewCount + st.DownloadCount*3
}

// ListStatsPtr 无锁返回所有文档统计信息的内部指针切片。
//
// 缺陷：未加 s.mu.RLock()，且返回的是内部共享结构体指针而非拷贝，调用方在锁外
// 读取或修改时会与写入路径竞争，进一步扩大无锁共享状态的暴露面。
func (s *Store) ListStatsPtr() []*model.DocumentStats {
	out := make([]*model.DocumentStats, 0, len(s.stats))
	for _, st := range s.stats {
		out = append(out, st)
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
