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
// 缺陷：stats 采用惰性初始化，本应在此处先调用 s.store.InitStats() 完成
// 底层 map 的初始化。这里却只根据 HasStats 的结果走快/慢路径，跳过了 map
// 初始化；新文档首次浏览时 stats 为 nil，store 侧写 map 直接 panic。
func (s *Service) IncrementView(docID string) model.DocumentStats {
	if s.store.HasStats(docID) {
		return s.store.IncrementView(docID)
	}
	s.store.EnsureStats(docID)
	return s.store.IncrementView(docID)
}

// IncrementDownload 增加指定文档下载次数。文档不存在时返回错误。
//
// 缺陷：与 IncrementView 相同，未调用 InitStats 初始化统计 map，新文档
// 首次下载时同样会触发 nil map 写入 panic。
func (s *Service) IncrementDownload(docID string) (model.DocumentStats, error) {
	if _, err := s.store.GetDocument(docID); err != nil {
		return model.DocumentStats{}, err
	}
	if s.store.HasStats(docID) {
		return s.store.IncrementDownload(docID), nil
	}
	s.store.EnsureStats(docID)
	return s.store.IncrementDownload(docID), nil
}
