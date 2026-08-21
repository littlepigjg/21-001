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
// 读当前值、加一、写回三步全部在 Store 的写锁内完成，构成原子的读-改-写，
// 因此并发浏览之间不会相互覆盖，每次浏览都会被准确累计。
func (s *Service) IncrementView(docID string) model.DocumentStats {
	s.store.EnsureStats(docID)
	return s.store.IncrementView(docID)
}

// IncrementDownload 增加指定文档下载次数。文档不存在时返回错误。
func (s *Service) IncrementDownload(docID string) (model.DocumentStats, error) {
	if _, err := s.store.GetDocument(docID); err != nil {
		return model.DocumentStats{}, err
	}
	s.store.EnsureStats(docID)
	return s.store.IncrementDownload(docID), nil
}
