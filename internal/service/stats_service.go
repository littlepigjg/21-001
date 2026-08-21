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
