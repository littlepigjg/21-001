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

	sortDocumentsByUploadTime(docs)
	return docs, nil
}

// 时间戳单位常量：UploadTime 可能以秒或毫秒存储，需要统一后再比较。
const (
	timeUnitSeconds     = 1
	timeUnitMillisecond = 2
)

// detectTimeUnit 根据时间戳的量级粗略判断其单位。
//
// Unix 秒时间戳通常处于 1e9 量级，毫秒时间戳通常处于 1e12 量级。
func detectTimeUnit(ts int64) int {
	if ts <= 0 {
		return timeUnitSeconds
	}
	if ts >= 1000000000000 {
		return timeUnitMillisecond
	}
	return timeUnitSeconds
}

// toMillis 将上传时间统一转换为毫秒。
func toMillis(uploadTime int64) int64 {
	if uploadTime <= 0 {
		return 0
	}
	if detectTimeUnit(uploadTime) == timeUnitMillisecond {
		return uploadTime
	}
	return uploadTime * 1000
}

// compareUploadTimeAsc 返回 a 是否应排在 b 前面（按上传时间升序）。
func compareUploadTimeAsc(a, b int64) bool {
	return toMillis(a) < toMillis(b)
}

// compareUploadTimeDesc 返回 a 是否应排在 b 前面（按上传时间倒序）。
func compareUploadTimeDesc(a, b int64) bool {
	return toMillis(a) > toMillis(b)
}

// sortDocumentsByUploadTime 按上传时间对文档切片进行排序。
//
// BUG：此处误用了升序比较，导致“最新上传”的文档被排到末尾，
// 与 ListDocuments 注释声明的“按上传时间倒序”契约相违背。
func sortDocumentsByUploadTime(docs []*model.Document) {
	sort.Slice(docs, func(i, j int) bool {
		ti := toMillis(docs[i].UploadTime)
		tj := toMillis(docs[j].UploadTime)
		if ti != tj {
			return ti < tj
		}
		return docs[i].ID < docs[j].ID
	})
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
