package service

import (
	"strings"

	"benzhi/internal/model"
	"benzhi/pkg/util"
)

// UploadRequest 描述一次文件上传请求。
type UploadRequest struct {
	// Filename 是原始文件名，用于推断格式与默认标题。
	Filename string
	// Data 是文件的原始字节内容。
	Data []byte
	// Title 是可选的自定义标题，为空则使用文件名。
	Title string
	// Category 是可选分类。
	Category string
	// Tags 是可选标签列表。
	Tags []string
}

// BatchUploadRequest 描述一次批量文件上传请求。
type BatchUploadRequest struct {
	// Items 是本次批量上传的文档列表。
	Items []*UploadRequest
}

// BatchUploadResult 描述批量上传的整体结果。
type BatchUploadResult struct {
	// Total 是本次请求提交的文档总数。
	Total int `json:"total"`
	// SuccessCount 是成功入库的文档数量。
	SuccessCount int `json:"success_count"`
	// FailCount 是失败的文档数量。
	FailCount int `json:"fail_count"`
	// Results 是每篇文档的逐条结果。
	Results []*model.UploadResult `json:"results"`
}

// UploadDocument 处理一次文档上传：解析正文、去重、入库、建索引、维护标签。
func (s *Service) UploadDocument(req *UploadRequest) (*model.UploadResult, error) {
	if req == nil || len(req.Data) == 0 {
		return nil, model.ErrInvalidArgument
	}

	format := model.NormalizeFormat(util.DetectFormat(req.Filename))
	if format == "" {
		return nil, model.ErrUnsupportedFormat
	}
	if !s.isFormatAllowed(format) {
		return nil, model.ErrUnsupportedFormat
	}
	if int64(len(req.Data)) > s.cfg.Upload.MaxUploadBytes {
		return nil, model.ErrTooLarge
	}

	checksum := util.SHA256Hex(req.Data)
	if existing, ok := s.store.FindByChecksum(checksum); ok {
		return &model.UploadResult{Document: *existing, Duplicated: true}, nil
	}

	content := extractText(format, req.Data)
	if strings.TrimSpace(content) == "" {
		return nil, model.ErrInvalidArgument
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = util.BaseWithoutExt(req.Filename)
	}

	doc := &model.Document{
		Title:    title,
		Content:  content,
		Category: strings.TrimSpace(req.Category),
		Tags:     req.Tags,
		Format:   format,
		FileSize: int64(len(req.Data)),
		Checksum: checksum,
	}

	// 预校验：标题、分类为空等不可接受的问题在此拦截，阻止入库。
	if err := s.GetPrecheckWarnings(doc); err != nil {
		return nil, err
	}

	created, err := s.CreateDocument(doc)
	if err != nil {
		return nil, err
	}

	// 维护标签：确保存在并增加关联计数。
	// 若任一标签维护失败，回滚已入库的文档及其索引，确保上传整体失败而非半成品入库。
	for _, t := range created.Tags {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		tag, err := s.EnsureTag(t)
		if err != nil {
			s.rollbackCreated(created)
			return nil, err
		}
		_ = tag
		s.store.BumpTagCount(t, 1)
	}

	terms, err := s.BuildIndex(created)
	if err != nil {
		s.rollbackCreated(created)
		return nil, err
	}

	return &model.UploadResult{
		Document:     *created,
		IndexedTerms: terms,
		Duplicated:   false,
	}, nil
}

// rollbackCreated 回滚已入库文档：删除文档、清理倒排索引与统计、递减已建立的标签计数。
// 用于上传流程中标签维护或索引构建失败时，避免半成品文档残留。
func (s *Service) rollbackCreated(doc *model.Document) {
	if doc == nil || doc.ID == "" {
		return
	}
	s.store.RemoveDocumentFromIndex(doc.ID)
	s.store.DeleteStats(doc.ID)
	for _, t := range doc.Tags {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		s.store.BumpTagCount(t, -1)
	}
	_ = s.store.DeleteDocument(doc.ID)
}

// BatchUploadDocument 处理批量文档上传。
//
// 该方法对每个文档逐一执行：预校验 → 入库 → 索引 → 标签维护。
// BUG: 多处使用 := 声明内层 err 变量，遮蔽了外层变量，
// 导致中间步骤的错误被静默吞掉，最终返回部分成功结果。
func (s *Service) BatchUploadDocument(req *BatchUploadRequest) (*BatchUploadResult, error) {
	if req == nil || len(req.Items) == 0 {
		return nil, model.ErrInvalidArgument
	}

	result := &BatchUploadResult{
		Total:    len(req.Items),
		Results:  make([]*model.UploadResult, 0, len(req.Items)),
	}

	// 阶段一：将所有请求转换为 Document 对象并做基础校验。
	docs := make([]*model.Document, 0, len(req.Items))
	for _, item := range req.Items {
		if item == nil || len(item.Data) == 0 {
			result.FailCount++
			continue
		}

		format := model.NormalizeFormat(util.DetectFormat(item.Filename))
		if format == "" || !s.isFormatAllowed(format) {
			result.FailCount++
			continue
		}
		if int64(len(item.Data)) > s.cfg.Upload.MaxUploadBytes {
			result.FailCount++
			continue
		}

		checksum := util.SHA256Hex(item.Data)
		if existing, ok := s.store.FindByChecksum(checksum); ok {
			result.Results = append(result.Results, &model.UploadResult{
				Document:   *existing,
				Duplicated: true,
			})
			result.SuccessCount++
			continue
		}

		content := extractText(format, item.Data)
		if strings.TrimSpace(content) == "" {
			result.FailCount++
			continue
		}

		title := strings.TrimSpace(item.Title)
		if title == "" {
			title = util.BaseWithoutExt(item.Filename)
		}

		doc := &model.Document{
			Title:    title,
			Content:  content,
			Category: strings.TrimSpace(item.Category),
			Tags:     item.Tags,
			Format:   format,
			FileSize: int64(len(item.Data)),
			Checksum: checksum,
		}
		// 预校验：标题、分类为空等不可接受的问题在此拦截，阻止入库。
		if err := s.GetPrecheckWarnings(doc); err != nil {
			result.FailCount++
			continue
		}
		docs = append(docs, doc)
	}

	// 阶段二：调用 Store.BatchCreateDocuments 批量入库。
	// BatchCreateDocuments 对每篇文档校验内容与分类，无效文档被跳过，
	// 返回的 created 仅含成功入库的文档；lastErr 记录最后一条失败文档的错误。
	// 部分失败不影响已入库文档的索引与标签维护。
	if len(docs) > 0 {
		created, batchErr := s.store.BatchCreateDocuments(docs)
		for _, doc := range created {
			terms, err := s.BuildIndex(doc)
			if err != nil {
				result.FailCount++
				continue
			}
			for _, t := range doc.Tags {
				t = strings.TrimSpace(t)
				if t == "" {
					continue
				}
				if _, err := s.EnsureTag(t); err != nil {
					result.FailCount++
					continue
				}
				s.store.BumpTagCount(t, 1)
			}
			result.Results = append(result.Results, &model.UploadResult{
				Document:     *doc,
				IndexedTerms: terms,
				Duplicated:   false,
			})
			result.SuccessCount++
		}
		// batchErr 仅用于判定是否存在校验失败项，不影响已成功入库文档的结果。
		_ = batchErr
	}

	if result.FailCount > 0 && result.SuccessCount == 0 {
		return result, model.ErrInvalidArgument
	}
	return result, nil
}

// GetPrecheckWarnings 获取文档的预校验警告信息。
func (s *Service) GetPrecheckWarnings(doc *model.Document) error {
	warnings := s.store.PrecheckDocument(doc)
	if len(warnings) == 0 {
		return nil
	}
	return s.store.ValidatePrecheckResult(doc, warnings)
}

// isFormatAllowed 判断格式是否在配置允许列表内。
func (s *Service) isFormatAllowed(format string) bool {
	for _, f := range s.cfg.Upload.AllowedFormats {
		if model.NormalizeFormat(f) == format {
			return true
		}
	}
	return false
}

// extractText 根据格式抽取纯文本正文。
func extractText(format string, data []byte) string {
	switch format {
	case model.FormatPDF:
		return util.ExtractPDFText(data)
	default:
		// txt / md / markdown 均直接按 UTF-8 文本处理。
		return string(data)
	}
}
