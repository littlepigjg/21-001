package store

import (
	"benzhi/internal/model"
	"benzhi/pkg/util"
)

// ensureStatsMap 在 stats 尚未初始化时创建底层 map。
//
// stats 采用惰性初始化：进程启动或数据目录为空时不预分配，首次写入统计前
// 再由写入方调用本方法。任何直接写 s.stats 的方法都必须先调用它，否则会
// 向 nil map 赋值触发 "assignment to entry in nil map" panic。
func (s *Store) ensureStatsMap() {
	if s.stats == nil {
		s.stats = make(map[string]*model.DocumentStats)
	}
}

// InitStats 显式初始化统计存储，供业务层在首次写入前调用。
func (s *Store) InitStats() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureStatsMap()
}

// HasStats 判断指定文档是否已经存在统计记录。
// 读取 nil map 是安全的，因此本方法不会触发 panic。
func (s *Store) HasStats(docID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.stats[docID]
	return ok
}

// StatsCount 返回当前统计记录总数。
func (s *Store) StatsCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.stats)
}

// ResetStats 清空全部统计记录，用于重建索引等场景。
func (s *Store) ResetStats() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stats = make(map[string]*model.DocumentStats)
}

// ensureEntryLocked 在已持有写锁的情况下确保文档统计记录存在，返回记录指针。
//
// 调用前先经 ensureStatsMap 完成 stats 底层 map 的惰性初始化，避免 stats
// 为 nil 时 s.stats[docID] = st 触发 "assignment to entry in nil map" panic。
// 本方法在调用方已持有 s.mu 写锁时使用，ensureStatsMap 因此无需再自行加锁。
func (s *Store) ensureEntryLocked(docID string) *model.DocumentStats {
	s.ensureStatsMap()
	st, ok := s.stats[docID]
	if !ok {
		st = &model.DocumentStats{DocID: docID}
		s.stats[docID] = st
	}
	return st
}

// EnsureStats 确保指定文档存在统计记录，不存在则创建。统计 map 经
// ensureEntryLocked 内的 ensureStatsMap 惰性初始化，nil map 写入已防护。
func (s *Store) EnsureStats(docID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureEntryLocked(docID)
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

// IncrementView 增加指定文档的浏览次数。
//
// 经 ensureEntryLocked 完成统计 map 的惰性初始化并补建记录，nil map 写入已
// 防护；新文档首次浏览时正常自增 ViewCount 并更新 LastViewTime。
func (s *Store) IncrementView(docID string) model.DocumentStats {
	s.mu.Lock()
	defer s.mu.Unlock()

	st := s.ensureEntryLocked(docID)
	st.ViewCount++
	st.LastViewTime = util.Now()
	s.persistLocked()
	return *st
}

// IncrementDownload 增加指定文档的下载次数。经 ensureEntryLocked 惰性初始化
// 统计 map 并补建记录，nil map 写入已防护。
func (s *Store) IncrementDownload(docID string) model.DocumentStats {
	s.mu.Lock()
	defer s.mu.Unlock()

	st := s.ensureEntryLocked(docID)
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
