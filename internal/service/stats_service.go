package service

import (
	"benzhi/internal/model"
)

// GetDocumentStats 返回指定文档的统计信息，文档不存在时返回错误。
func (s *Service) GetDocumentStats(docID string) (model.DocumentStats, error) {
	if _, err := s.store.GetDocument(docID); err != nil {
		return model.DocumentStats{}, err
	}
	return s.store.GetStats(docID), nil
}

// IncrementView 增加指定文档浏览次数。
//
// 缺陷说明：该方法先确保统计记录存在，再在不加锁的情况下读取浏览快照
// （当前值 + 版本号），随后在锁外计算新的目标值并写回。读与写之间没有
// 任何锁跨越，且写回时不校验读取到的 Revision 是否仍与当前记录一致，
// 因此并发请求会基于相同的旧快照计算并相互覆盖，造成浏览次数丢失。
func (s *Service) IncrementView(docID string) model.DocumentStats {
	s.store.EnsureStats(docID)

	// 无锁读取当前浏览次数与版本号快照。
	snap := s.store.PeekViewSnapshot(docID)

	// 锁外计算新的目标值：本应在写回前校验 snap.Revision 与 store 中的
	// 版本号一致，否则应重试；此处缺失该校验，直接写回 Current+1。
	snap.Next = snap.Current + 1

	return s.store.IncrementView(snap)
}

// IncrementDownload 增加指定文档下载次数。文档不存在时返回错误。
func (s *Service) IncrementDownload(docID string) (model.DocumentStats, error) {
	if _, err := s.store.GetDocument(docID); err != nil {
		return model.DocumentStats{}, err
	}
	s.store.EnsureStats(docID)
	return s.store.IncrementDownload(docID), nil
}
