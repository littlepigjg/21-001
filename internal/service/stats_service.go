package service

import (
	"benzhi/internal/model"
)

func (s *Service) GetDocumentStats(docID string) (model.DocumentStats, error) {
	if _, err := s.store.GetDocument(docID); err != nil {
		return model.DocumentStats{}, err
	}
	return s.store.GetStats(docID), nil
}

func (s *Service) IncrementView(docID string) model.DocumentStats {
	s.store.EnsureStats(docID)
	return s.store.IncrementView(docID)
}

func (s *Service) IncrementDownload(docID string) (model.DocumentStats, error) {
	if _, err := s.store.GetDocument(docID); err != nil {
		return model.DocumentStats{}, err
	}
	s.store.EnsureStats(docID)
	return s.store.IncrementDownload(docID), nil
}
