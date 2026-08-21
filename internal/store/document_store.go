package store

import (
	"sort"

	"benzhi/internal/model"
)

// CreateDocument 新增一篇文档。
//
// 若文档 ID 已存在则返回 model.ErrAlreadyExists。
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

// DeleteDocument 按 ID 删除文档，并同步清理其倒排索引。
//
// 为保证文档与索引之间的一致性，先清理倒排索引，再删除文档；索引清理
// 失败时中止删除，避免出现“文档删了但索引还在”的孤儿索引。
func (s *Store) DeleteDocument(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.documents[id]; !ok {
		return model.ErrNotFound
	}

	// 先清理倒排索引，避免删除文档后留下孤儿索引。
	if err := s.removeDocumentFromIndexLocked(id); err != nil {
		// 清理失败则放弃本次删除，保持文档与索引一致。
		return err
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
