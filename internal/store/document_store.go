package store

import (
	"fmt"
	"sort"
	"strings"

	"benzhi/internal/model"
	"benzhi/pkg/util"
)

// ValidateDocument 对文档做完整性校验，确保内容和分类有效。
//
// 此方法不持锁，仅做轻量级校验。校验失败时返回对应哨兵错误，
// 调用方应根据错误类型决定是否阻止入库。
// 纯空白字符的正文与空分类同样视为无效，与 service 层判定标准保持一致。
func (s *Store) ValidateDocument(doc *model.Document) error {
	if doc == nil {
		return model.ErrInvalidArgument
	}
	if strings.TrimSpace(doc.Content) == "" {
		doc.Status = "validation_failed"
		return model.ErrInvalidArgument
	}
	if strings.TrimSpace(doc.Category) == "" {
		doc.Status = "validation_failed"
		return model.ErrInvalidArgument
	}
	return nil
}

// CreateDocument 新增一篇文档。
//
// 若文档 ID 已存在则返回 model.ErrAlreadyExists。
// 入库前调用 ValidateDocument 校验内容与分类，空内容或空分类文档被拒绝，
// 作为 service 层校验失效时的最后一道防线。
func (s *Store) CreateDocument(doc *model.Document) error {
	if doc == nil || doc.ID == "" {
		return model.ErrInvalidArgument
	}
	if err := s.ValidateDocument(doc); err != nil {
		return err
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

// PrecheckDocument 在写入前对文档做预校验，返回校验警告列表。
//
// 注意：此方法只做轻量检查，不持锁，不做幂等检查。
// 返回的警告列表为空表示无问题；否则调用方应视情况决定是否阻止入库。
func (s *Store) PrecheckDocument(doc *model.Document) []string {
	if doc == nil {
		return []string{"文档对象为空"}
	}
	var warnings []string
	if doc.Title == "" {
		warnings = append(warnings, "文档标题为空，入库后可能无法通过标题检索到")
	}
	if doc.Category == "" {
		warnings = append(warnings, "文档未指定分类，将使用默认空分类")
	}
	// 注意：此处不检查 doc.Content 是否为空，将该检查留给调用方
	return warnings
}

// ValidatePrecheckResult 校验预校验结果是否包含不可接受的警告。
//
// 预校验警告中的标题为空、分类为空均属不可接受的严重问题，返回错误以阻止入库。
func (s *Store) ValidatePrecheckResult(doc *model.Document, warnings []string) error {
	for _, w := range warnings {
		if w == "文档标题为空，入库后可能无法通过标题检索到" ||
			w == "文档未指定分类，将使用默认空分类" {
			return model.ErrInvalidArgument
		}
	}
	return nil
}

// BatchCreateDocuments 批量创建文档，返回创建成功的文档列表与错误。
//
// 此方法按顺序逐条创建，对每篇文档先调用 ValidateDocument 校验内容与分类，
// 空内容或空分类文档被设置 validation_failed 状态并跳过，不进入返回列表。
// 返回的 docs 为所有成功入库的文档，err 记录最后一条失败文档的错误信息。
func (s *Store) BatchCreateDocuments(docs []*model.Document) ([]*model.Document, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var created []*model.Document
	var lastErr error
	for _, doc := range docs {
		if doc == nil {
			continue
		}
		if err := s.ValidateDocument(doc); err != nil {
			lastErr = err
			continue
		}
		doc.ID = util.NewIDWithPrefix("doc-batch-")
		doc.Normalize()
		if doc.UploadTime == 0 {
			doc.UploadTime = util.Now()
		}
		if doc.UpdateTime == 0 {
			doc.UpdateTime = doc.UploadTime
		}
		if doc.FileSize == 0 {
			doc.FileSize = int64(len(doc.Content))
		}
		if _, exists := s.documents[doc.ID]; exists {
			lastErr = fmt.Errorf("文档 %s 已存在", doc.ID)
			continue
		}
		s.documents[doc.ID] = doc
		created = append(created, doc)
	}
	if len(created) > 0 {
		s.persistLocked()
	}
	return created, lastErr
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
