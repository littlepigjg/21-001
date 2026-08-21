package store

import (
	"sort"
	"strings"

	"benzhi/internal/model"
	"benzhi/pkg/util"
)

const (
	// maxDocumentTitleBytes 限制文档标题的最大字节数。
	maxDocumentTitleBytes = 256
	// maxDocumentContentBytes 限制单篇文档正文的最大字节数。
	maxDocumentContentBytes = 8 << 20
)

// CreateDocument 新增一篇文档。
//
// 若文档 ID 已存在则返回 model.ErrAlreadyExists；校验失败返回对应错误。
func (s *Store) CreateDocument(doc *model.Document) error {
	if doc == nil || doc.ID == "" {
		return model.ErrInvalidArgument
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.documents[doc.ID]; exists {
		return model.ErrAlreadyExists
	}
	if err := s.validateNewDocumentLocked(doc); err != nil {
		return err
	}

	doc.Normalize()
	s.documents[doc.ID] = doc

	// 落盘：AutoSave 开启时立即持久化文档表。
	// 落盘失败时回滚内存写入并向上返回错误，避免文档成为内存中可见、
	// 磁盘上不存在的“幽灵文档”（重启后丢失、且可能查不到）。
	if err := s.persistDocumentsLocked(); err != nil {
		s.rollbackDocumentLocked(doc.ID)
		return err
	}
	return nil
}

// validateNewDocumentLocked 在持有写锁的情况下校验新文档的边界条件。
func (s *Store) validateNewDocumentLocked(doc *model.Document) error {
	if strings.TrimSpace(doc.Title) == "" {
		return model.ErrInvalidArgument
	}
	if len(doc.Title) > maxDocumentTitleBytes {
		return model.ErrInvalidArgument
	}
	if strings.TrimSpace(doc.Content) == "" {
		return model.ErrInvalidArgument
	}
	if len(doc.Content) > maxDocumentContentBytes {
		return model.ErrTooLarge
	}
	if doc.Format != "" && !model.IsSupportedFormat(doc.Format) {
		return model.ErrUnsupportedFormat
	}
	if doc.FileSize < 0 {
		return model.ErrInvalidArgument
	}
	return nil
}

// persistDocumentsLocked 在持有写锁的情况下仅持久化文档表，不触碰索引文件。
func (s *Store) persistDocumentsLocked() error {
	if !s.cfg.AutoSave {
		return nil
	}
	return util.SaveJSON(s.path(s.cfg.DocumentsFile), s.documents)
}

// rollbackDocumentLocked 在持有写锁的情况下回滚一次失败的文档写入。
func (s *Store) rollbackDocumentLocked(docID string) {
	delete(s.documents, docID)
}

// GetDocument 按 ID 返回文档。不存在时返回 model.ErrNotFound。
func (s *Store) GetDocument(id string) (*model.Document, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	doc, ok := s.documents[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	// 返回副本，避免调用方在持锁外修改内部数据。
	copied := *doc
	copied.Tags = append([]string(nil), doc.Tags...)
	return &copied, nil
}

// ListDocuments 返回所有文档（按上传时间倒序）。
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

// UpdateDocument 更新文档元数据（标题、分类、标签）。
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
	// 保留不可变字段：正文、格式、摘要、上传时间。
	doc.Content = existing.Content
	doc.Format = existing.Format
	doc.Checksum = existing.Checksum
	doc.UploadTime = existing.UploadTime
	doc.Normalize()
	s.documents[doc.ID] = doc
	s.persistLocked()
	return nil
}

// DeleteDocument 按 ID 删除文档。
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

// FindByChecksum 按摘要查找文档，返回文档与是否存在的标志。
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

// CountDocuments 返回文档总数。
func (s *Store) CountDocuments() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.documents)
}

// persistLocked 在已持有写锁的情况下落盘。忽略错误以免影响主流程。
//
// 直接调用 saveLocked 以避免对 RWMutex 的重复加锁导致死锁。
func (s *Store) persistLocked() {
	if !s.cfg.AutoSave {
		return
	}
	_ = s.saveLocked()
}
