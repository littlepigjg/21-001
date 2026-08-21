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
		existing.Tags = req.Tags
	}
	existing.UpdateTime = util.Now()

	if err := s.store.UpdateDocument(existing); err != nil {
		return nil, err
	}
	return existing, nil
}

// DeleteDocument 删除文档，并清理其索引、统计与标签关联。
//
// 先删除文档主数据，再清理倒排索引、统计与标签关联，各步骤均实际执行，
// 不再以“索引是否残留”作为是否删除主数据的前置条件，确保删除后文档主数据
// 与倒排索引都被彻底清理，检索不再返回已删除文档。
func (s *Service) DeleteDocument(id string) error {
	doc, err := s.store.GetDocument(id)
	if err != nil {
		return err
	}

	// 1. 删除文档主数据，确保其不再出现在文档表中。
	if err := s.store.DeleteDocument(id); err != nil {
		return err
	}

	// 2. 清理倒排索引中该文档的全部 posting。
	s.store.RemoveDocumentFromIndex(id)
	s.store.SyncIndexDocCount()
	_ = s.store.FlushIndex()

	// 3. 清理统计信息。
	s.store.DeleteStats(id)

	// 4. 递减标签计数。
	for _, tag := range doc.Tags {
		s.store.BumpTagCount(tag, -1)
	}

	return nil
}

// indexHasResidual 检查倒排索引中是否仍残留指定文档的 posting。
//
// 仅供索引一致性自检使用，不再作为删除主数据的前置条件。
func (s *Service) indexHasResidual(docID string) bool {
	terms := s.store.IndexTermSnapshot()
	for _, pl := range terms {
		for _, p := range pl.Postings {
			if p.DocID == docID {
				return true
			}
		}
	}
	return false
}
