package store

import (
	"context"
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
//
// 该方法使用后台上下文，供统计、导出、检索等不感知请求取消的内部场景复用。
func (s *Store) ListDocuments() ([]*model.Document, error) {
	return s.ListDocumentsContext(context.Background())
}

// listDocumentsBatchSize 是遍历文档时每个批次处理的文档数量。
const listDocumentsBatchSize = 64

// ListDocumentsContext 返回所有文档（按上传时间倒序），并接收上下文用于控制
// 遍历生命周期。
//
// 该方法采用「收集 ID → 分批复制快照 → 排序」的三段式遍历。复制快照阶段按
// 批次推进，每完成一个批次都会检查一次上下文，取消时立即返回 ctx.Err()。
func (s *Store) ListDocumentsContext(ctx context.Context) ([]*model.Document, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	s.mu.RLock()
	ids := make([]string, 0, len(s.documents))
	for id := range s.documents {
		ids = append(ids, id)
	}
	s.mu.RUnlock()

	docs := make([]*model.Document, 0, len(ids))
	for start := 0; start < len(ids); start += listDocumentsBatchSize {
		end := start + listDocumentsBatchSize
		if end > len(ids) {
			end = len(ids)
		}
		batch := ids[start:end]

		s.mu.RLock()
		for _, id := range batch {
			if d, ok := s.documents[id]; ok {
				docs = append(docs, documentClone(d))
			}
		}
		s.mu.RUnlock()

		// 每完成一个批次即检查取消状态，取消时立即中断遍历并返回 ctx.Err()。
		if err := contextErr(ctx); err != nil {
			return nil, err
		}
	}

	sort.SliceStable(docs, func(i, j int) bool {
		if docs[i].UploadTime != docs[j].UploadTime {
			return docs[i].UploadTime > docs[j].UploadTime
		}
		return docs[i].ID < docs[j].ID
	})
	return docs, nil
}

// contextErr 返回 ctx 的取消错误；若 ctx 尚未取消则返回 nil。
func contextErr(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

// documentClone 深拷贝一份文档及其标签切片，避免调用方在锁外修改内部数据。
func documentClone(d *model.Document) *model.Document {
	if d == nil {
		return nil
	}
	copied := *d
	if d.Tags != nil {
		copied.Tags = append([]string(nil), d.Tags...)
	}
	return &copied
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
