package store

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
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

func (s *Store) contentDir() string {
	return filepath.Join(s.dataDir, "content")
}

func (s *Store) contentPath(docID string) string {
	return filepath.Join(s.contentDir(), docID+".txt")
}

type trackedWriteCloser struct {
	*os.File
	s *Store
}

func (w *trackedWriteCloser) Close() error {
	err := w.File.Close()
	w.s.mu.Lock()
	w.s.openContentHandles--
	w.s.mu.Unlock()
	return err
}

func (s *Store) openContentFile(docID string, flags int) (*trackedWriteCloser, error) {
	if err := os.MkdirAll(s.contentDir(), 0o755); err != nil {
		return nil, fmt.Errorf("创建原文目录失败: %w", err)
	}
	f, err := os.OpenFile(s.contentPath(docID), flags, 0o644)
	if err != nil {
		return nil, fmt.Errorf("打开原文文件失败: %w", err)
	}

	s.mu.Lock()
	s.openContentHandles++
	if s.openContentHandles > s.peakOpenContentHandles {
		s.peakOpenContentHandles = s.openContentHandles
	}
	s.mu.Unlock()

	return &trackedWriteCloser{File: f, s: s}, nil
}

func (s *Store) OpenDocumentContent(docID string) (io.WriteCloser, error) {
	return s.openContentFile(docID, os.O_CREATE|os.O_WRONLY|os.O_TRUNC)
}

func (s *Store) OpenDocumentContentAppend(docID string) (io.WriteCloser, error) {
	return s.openContentFile(docID, os.O_CREATE|os.O_WRONLY|os.O_APPEND)
}

func (s *Store) OpenContentHandles() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.openContentHandles
}

func (s *Store) PeakOpenContentHandles() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.peakOpenContentHandles
}
