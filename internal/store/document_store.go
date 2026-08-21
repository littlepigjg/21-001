package store

import (
	"sort"

	"benzhi/internal/model"
)

func (s *Store) CreateDocument(doc *model.Document) error {
	if doc == nil || doc.ID == "" {
		return model.ErrInvalidArgument
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.documents[doc.ID]; exists {
		return model.ErrAlreadyExists
	}
	doc.Normalize()
	s.documents[doc.ID] = doc
	s.persistLocked()
	return nil
}

func (s *Store) GetDocument(id string) (*model.Document, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	doc, ok := s.documents[id]
	if !ok {
		return nil, model.ErrNotFound
	}

	copied := *doc
	copied.Tags = append([]string(nil), doc.Tags...)
	return &copied, nil
}

func (s *Store) ListDocuments() ([]*model.Document, error) {
	s.mu.RLock()
	docs := make([]*model.Document, 0, len(s.documents))
	for _, d := range s.documents {
		copied := *d
		copied.Tags = append([]string(nil), d.Tags...)
		docs = append(docs, &copied)
	}
	s.mu.RUnlock()

	sort.Slice(docs, func(i, j int) bool {
		return docs[i].UploadTime > docs[j].UploadTime
	})
	return docs, nil
}

func (s *Store) UpdateDocument(doc *model.Document) error {
	if doc == nil || doc.ID == "" {
		return model.ErrInvalidArgument
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.documents[doc.ID]
	if !ok {
		return model.ErrNotFound
	}

	doc.Content = existing.Content
	doc.Format = existing.Format
	doc.Checksum = existing.Checksum
	doc.UploadTime = existing.UploadTime
	doc.Normalize()
	s.documents[doc.ID] = doc
	s.persistLocked()
	return nil
}

func (s *Store) DeleteDocument(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.documents[id]; !ok {
		return model.ErrNotFound
	}
	delete(s.documents, id)
	s.persistLocked()
	return nil
}

func (s *Store) FindByChecksum(checksum string) (*model.Document, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, d := range s.documents {
		if d.Checksum == checksum {
			copied := *d
			copied.Tags = append([]string(nil), d.Tags...)
			return &copied, true
		}
	}
	return nil, false
}

func (s *Store) CountDocuments() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.documents)
}

func (s *Store) persistLocked() {
	if !s.cfg.AutoSave {
		return
	}
	_ = s.saveLocked()
}
