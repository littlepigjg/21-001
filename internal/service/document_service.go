package service

import (
	"benzhi/internal/model"
	"benzhi/pkg/util"
)

func (s *Service) CreateDocument(doc *model.Document) (*model.Document, error) {
	if doc == nil {
		return nil, model.ErrInvalidArgument
	}
	if doc.Title == "" || doc.Content == "" {
		return nil, model.ErrInvalidArgument
	}

	doc.ID = util.NewIDWithPrefix("doc-")
	doc.Normalize()
	if doc.UploadTime == 0 {
		doc.UploadTime = util.Now()
	}
	if doc.UpdateTime == 0 {
		doc.UpdateTime = doc.UploadTime
	}

	if err := s.store.CreateDocument(doc); err != nil {
		return nil, err
	}
	return doc, nil
}

func (s *Service) GetDocument(id string) (*model.Document, error) {
	doc, err := s.store.GetDocument(id)
	if err != nil {
		return nil, err
	}

	s.store.EnsureStats(id)
	s.store.IncrementView(id)
	return doc, nil
}

func (s *Service) ListDocuments(page, pageSize int) ([]*model.Document, int, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = s.cfg.Search.DefaultPageSize
	}
	if pageSize > s.cfg.Search.MaxPageSize {
		pageSize = s.cfg.Search.MaxPageSize
	}

	docs, err := s.store.ListDocuments()
	if err != nil {
		return nil, 0, err
	}

	total := len(docs)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return docs[start:end], total, nil
}

func (s *Service) UpdateDocument(id string, req *model.Document) (*model.Document, error) {
	if req == nil {
		return nil, model.ErrInvalidArgument
	}
	existing, err := s.store.GetDocument(id)
	if err != nil {
		return nil, err
	}

	if req.Title != "" {
		existing.Title = req.Title
	}
	if req.Category != "" {
		existing.Category = req.Category
	}
	if req.Tags != nil {
		existing.Tags = req.Tags
	}
	existing.UpdateTime = util.Now()

	if err := s.store.UpdateDocument(existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *Service) DeleteDocument(id string) error {
	doc, err := s.store.GetDocument(id)
	if err != nil {
		return err
	}

	s.store.RemoveDocumentFromIndex(id)

	s.store.DeleteStats(id)

	for _, tag := range doc.Tags {
		s.store.BumpTagCount(tag, -1)
	}

	return s.store.DeleteDocument(id)
}
