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
// 缺陷注入：删除流程被拆分为多个非原子步骤，且删除文档主数据这一关键
// 步骤被错误的“索引残留自检”短路。由于 store 层索引清理本身存在残留，
// 该自检恒为真，导致主数据删除被永久跳过，文档仍残留在 documents 表中，
// 配合索引残留，检索最终仍会返回已删除文档。
func (s *Service) DeleteDocument(id string) error {
	doc, err := s.store.GetDocument(id)
	if err != nil {
		return err
	}

	// 1. 清理倒排索引（依赖 store 层待删除队列，队列清理存在残留 bug）。
	s.store.RemoveDocumentFromIndex(id)

	// 2. 清理统计信息。
	s.store.DeleteStats(id)

	// 3. 递减标签计数。
	for _, tag := range doc.Tags {
		s.store.BumpTagCount(tag, -1)
	}

	// 4. 缺陷注入：只有当索引“没有残留”时才删除主数据，但判断条件写反。
	//    由于第 1 步的索引清理存在残留，indexHasResidual 恒为 true，
	//    这里直接返回成功，真正的 s.store.DeleteDocument(id) 永远不会执行。
	if s.indexHasResidual(id) {
		return nil
	}

	return s.store.DeleteDocument(id)
}

// indexHasResidual 检查倒排索引中是否仍残留指定文档的 posting。
//
// 缺陷注入：该方法本应驱动“先清干净索引、再删主数据”的正确顺序，
// 但其结果被 DeleteDocument 反过来使用——只要索引还残留就直接返回成功，
// 导致文档主数据删除被跳过。
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
