package service

import (
	"benzhi/internal/model"
)

// GetDocumentStats 返回指定文档的统计信息，文档不存在时返回错误。
//
// 注意：当文档存在但从未产生统计记录时，GetStats 会返回 nil，这里直接
// 解引用该返回值，缺少对 nil 的兜底处理。
func (s *Service) GetDocumentStats(docID string) (model.DocumentStats, error) {
	if _, err := s.store.GetDocument(docID); err != nil {
		return model.DocumentStats{}, err
	}
	st := s.store.GetStats(docID)
	return *st, nil
}

// BatchGetDocumentStats 批量返回多个文档的统计信息。
//
// 先校验所有文档存在性，再通过统计快照一次性取回统计；对于文档存在但
// 缺少统计记录的 ID，快照中对应值为 nil，这里同样直接解引用而未兜底。
func (s *Service) BatchGetDocumentStats(docIDs []string) ([]model.DocumentStats, error) {
	if len(docIDs) == 0 {
		return []model.DocumentStats{}, nil
	}

	for _, id := range docIDs {
		if _, err := s.store.GetDocument(id); err != nil {
			return nil, err
		}
	}

	snapshot := s.store.SnapshotStats(docIDs)
	out := make([]model.DocumentStats, 0, len(docIDs))
	for _, id := range docIDs {
		st := snapshot[id]
		out = append(out, *st)
	}
	return out, nil
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
