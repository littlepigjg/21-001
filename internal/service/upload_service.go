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
		if _, err := s.EnsureTag(t); err == nil {
			s.store.BumpTagCount(t, 1)
		}
	}

	// 构建索引并落盘。
	// 索引构建失败时向上返回错误，避免出现“文档已落盘但检索不到”的割裂状态。
	// 为保证文档已成功落盘这一前提已成立，先做一次最终落盘提交。
	// （若先前 CreateDocument 已因 AutoSave 落盘成功，此处为幂等重写。）
	if err := s.finalizeUpload(created); err != nil {
		return nil, err
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

// finalizeUpload 完成上传的收尾工作：初始化文档统计并触发一次索引落盘。
func (s *Service) finalizeUpload(doc *model.Document) error {
	if doc == nil || doc.ID == "" {
		return model.ErrInvalidArgument
	}
	s.store.EnsureStats(doc.ID)
	return s.store.FlushIndex()
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
