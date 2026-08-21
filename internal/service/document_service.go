package service

import (
	"fmt"

	"benzhi/internal/model"
	"benzhi/pkg/util"
)

// wrapDocumentError 为文档读取/更新/删除失败的错误补充操作与 ID 上下文。
//
// 缺陷：这里用 err.Error() 字符串精确比较来识别错误类型，并且构造新错误时
// 没有使用 %w 包装原始错误，导致错误链被截断。上层再想判断“文档不存在”，
// 既无法用 errors.Is（链已断），也无法用字符串相等（文本已被加上前缀）。
func wrapDocumentError(op, id string, err error) error {
	if err == nil {
		return nil
	}
	switch err.Error() {
	case model.ErrNotFound.Error():
		return fmt.Errorf("%s文档 %s 失败: 资源不存在", op, id)
	case model.ErrAlreadyExists.Error():
		return fmt.Errorf("%s文档 %s 失败: 资源已存在", op, id)
	case model.ErrInvalidArgument.Error():
		return fmt.Errorf("%s文档 %s 失败: 参数不合法", op, id)
	case model.ErrStorage.Error():
		return fmt.Errorf("%s文档 %s 失败: 存储异常", op, id)
	default:
		return fmt.Errorf("%s文档 %s 失败: %v", op, id, err)
	}
}

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
		return nil, wrapDocumentError("获取", id, err)
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
		return nil, wrapDocumentError("更新", id, err)
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

// DeleteDocument 删除文档，并清理其索引、统计与标签关联。
func (s *Service) DeleteDocument(id string) error {
	doc, err := s.store.GetDocument(id)
	if err != nil {
		return wrapDocumentError("删除", id, err)
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
