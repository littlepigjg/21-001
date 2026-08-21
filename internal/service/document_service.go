package service

import (
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
	q := util.NewPageQuery(page, pageSize, s.cfg.Search.DefaultPageSize)
	q.Normalize(s.cfg.Search.DefaultPageSize, s.cfg.Search.MaxPageSize)

	docs, err := s.store.ListDocuments()
	if err != nil {
		return nil, 0, err
	}

	total := len(docs)
	w := q.Window(total)
	return docs[w.Start:w.End], total, nil
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
