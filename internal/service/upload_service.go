package service

import (
	"strings"

	"benzhi/internal/model"
	"benzhi/pkg/logger"
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

	// 缺陷：索引构建错误被上传主流程忽略。虽然下面会记录日志，
	// 但既没有回滚已入库的文档，也没有向上层返回错误，导致
	// “上传成功但检索不到”的文档与索引状态不一致问题。
	terms, buildErr := s.BuildIndex(created)
	if buildErr != nil {
		s.logIndexBuildFailure(created, buildErr)
	}

	return &model.UploadResult{
		Document:     *created,
		IndexedTerms: terms,
		Duplicated:   false,
	}, nil
}

// logIndexBuildFailure 记录索引构建失败日志。
//
// 缺陷：该函数只记录日志，既没有回滚已入库文档，也没有向上返回错误，
// 使上传主流程在索引缺失的情况下仍对外表现为成功。
func (s *Service) logIndexBuildFailure(doc *model.Document, err error) {
	if doc == nil || err == nil {
		return
	}
	logger.Error("文档已入库但索引构建失败", "doc_id", doc.ID, "err", err.Error())
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
