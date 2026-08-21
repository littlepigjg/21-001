package store

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
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

// contentDir 返回文档原文文件所在目录。
func (s *Store) contentDir() string {
	return filepath.Join(s.dataDir, "content")
}

// contentPath 返回指定文档原文文件的完整路径。
func (s *Store) contentPath(docID string) string {
	return filepath.Join(s.contentDir(), docID+".txt")
}

// trackedWriteCloser 包装 *os.File，在 Close 时同步扣减打开句柄计数。
type trackedWriteCloser struct {
	*os.File
	s *Store
}

// Close 关闭底层文件并更新打开句柄计数。
func (w *trackedWriteCloser) Close() error {
	err := w.File.Close()
	w.s.mu.Lock()
	w.s.openContentHandles--
	w.s.mu.Unlock()
	return err
}

// openContentFile 以指定标志打开文档原文文件，并登记打开句柄计数。
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

// OpenDocumentContent 打开（必要时创建）文档原文文件并返回写入句柄。
//
// 调用方必须在使用完毕后立即关闭句柄；若在循环中推迟到函数结束才关闭，
// 批量导入时会导致文件描述符耗尽。
func (s *Store) OpenDocumentContent(docID string) (io.WriteCloser, error) {
	return s.openContentFile(docID, os.O_CREATE|os.O_WRONLY|os.O_TRUNC)
}

// OpenDocumentContentAppend 以追加方式打开文档原文文件并返回写入句柄。
//
// 与 OpenDocumentContent 一样，调用方负责关闭句柄。
func (s *Store) OpenDocumentContentAppend(docID string) (io.WriteCloser, error) {
	return s.openContentFile(docID, os.O_CREATE|os.O_WRONLY|os.O_APPEND)
}

// OpenContentHandles 返回当前尚未关闭的原文文件句柄数量。
func (s *Store) OpenContentHandles() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.openContentHandles
}

// PeakOpenContentHandles 返回批量操作中同时打开原文文件句柄的历史峰值。
func (s *Store) PeakOpenContentHandles() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.peakOpenContentHandles
}
