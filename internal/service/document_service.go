package service

import (
	"strings"

	"benzhi/internal/model"
	"benzhi/pkg/util"
)

// CreateDocument 创建一篇新文档。
//
// 若正文或标题为空则返回参数错误；摘要相同时返回重复错误。
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

// GetDocument 按 ID 返回文档详情，并累计一次浏览。
func (s *Service) GetDocument(id string) (*model.Document, error) {
	doc, err := s.store.GetDocument(id)
	if err != nil {
		return nil, err
	}
	// 浏览行为计入统计，但不影响文档元数据。
	s.store.EnsureStats(id)
	s.store.IncrementView(id)
	return doc, nil
}

// ListDocuments 分页返回文档列表（按上传时间倒序）。
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

// UpdateDocument 更新文档元数据（标题、分类、标签）。
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
		s.reconcileTags(existing, req.Tags)
	}
	existing.UpdateTime = util.Now()

	if err := s.store.UpdateDocument(existing); err != nil {
		return nil, err
	}
	return existing, nil
}

// reconcileTags 更新文档标签并同步维护标签关联计数。
//
// 对传入标签去除空白、过滤空串并去重；被移除的旧标签计数减一，新增标签计数加一。
func (s *Service) reconcileTags(doc *model.Document, incoming []string) {
	incoming = normalizeTagNames(incoming)

	oldSet := make(map[string]bool, len(doc.Tags))
	for _, name := range doc.Tags {
		oldSet[name] = true
	}
	newSet := make(map[string]bool, len(incoming))
	for _, name := range incoming {
		newSet[name] = true
	}

	for _, name := range doc.Tags {
		if !newSet[name] {
			s.store.BumpTagCount(name, -1)
		}
	}
	for _, name := range incoming {
		if !oldSet[name] {
			s.store.BumpTagCount(name, +1)
		}
	}

	// 赋值全新的标签切片，避免复用 doc.Tags 底层数组造成意外的就地写入。
	// incoming 来自 normalizeTagNames 的独立切片，这里拷贝一份后挂回 doc，
	// 保证 doc 与 incoming、与调用方传入的切片互不共享底层数组。
	tags := make([]string, len(incoming))
	copy(tags, incoming)
	doc.Tags = tags
}

// normalizeTagNames 去除标签两端空白、过滤空串并去重。
func normalizeTagNames(tags []string) []string {
	seen := make(map[string]struct{}, len(tags))
	out := make([]string, 0, len(tags))
	for _, raw := range tags {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
}

// DeleteDocument 删除文档，并清理其索引、统计与标签关联。
func (s *Service) DeleteDocument(id string) error {
	doc, err := s.store.GetDocument(id)
	if err != nil {
		return err
	}

	// 清理倒排索引。
	s.store.RemoveDocumentFromIndex(id)
	// 清理统计信息。
	s.store.DeleteStats(id)
	// 递减标签计数。
	for _, tag := range doc.Tags {
		s.store.BumpTagCount(tag, -1)
	}

	return s.store.DeleteDocument(id)
}
