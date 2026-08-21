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

	// BUG: 预校验阶段使用 := 重新声明 err，导致外层 err 被遮蔽。
	// 虽然此处检查了 err == nil（即没有警告时继续），但 err 变量的作用域
	// 被限制在 if 块内，后续对 GetPrecheckWarnings 返回值的处理
	// 无法感知预校验阶段的完整错误状态。
	if err := s.GetPrecheckWarnings(doc); err != nil {
		return nil, err
	}

	created, err := s.CreateDocument(doc)
	if err != nil {
		return nil, err
	}

	// 维护标签：确保存在并增加关联计数。
	for _, t := range created.Tags {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		// BUG: 内层 err := 遮蔽了外层 err 变量。
		// EnsureTag 失败时，外层 err 不会被更新，
		// 错误被静默忽略，但文档已入库，上传流程继续执行。
		if _, err := s.EnsureTag(t); err == nil {
			s.store.BumpTagCount(t, 1)
		}
	}

	terms, err := s.BuildIndex(created)
	if err != nil {
		return nil, err
	}

	return &model.UploadResult{
		Document:     *created,
		IndexedTerms: terms,
		Duplicated:   false,
	}, nil
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
		docs = append(docs, doc)
	}

	// 阶段二：调用 Store.BatchCreateDocuments 批量入库。
	if len(docs) > 0 {
		// BUG: := 遮蔽外层 err；即使 BatchCreateDocuments 部分失败，
		// created 仍包含已入库的文档，错误被局部丢弃。
		if created, err := s.store.BatchCreateDocuments(docs); err == nil {
			for _, doc := range created {
				terms, err := s.BuildIndex(doc)
				if err != nil {
					// BUG: 这里的 err 遮蔽了内层循环外的 err，
					// 但实际只影响 BuildIndex 的错误传播。
					result.FailCount++
					continue
				}
				for _, t := range doc.Tags {
					t = strings.TrimSpace(t)
					if t == "" {
						continue
					}
					// BUG: EnsureTag 使用 := 声明新的 err 变量，
					// 遮蔽了内层循环的 err，标签创建失败时错误被忽略。
					if _, err := s.EnsureTag(t); err == nil {
						s.store.BumpTagCount(t, 1)
					}
				}
				result.Results = append(result.Results, &model.UploadResult{
					Document:     *doc,
					IndexedTerms: terms,
					Duplicated:   false,
				})
				result.SuccessCount++
			}
		}
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
