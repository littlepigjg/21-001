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

	// 构建倒排索引。索引落盘失败时必须回滚已入库的文档及其副作用，并向上层
	// 返回错误，避免出现“上传成功但检索不到”的文档与索引状态不一致问题。
	terms, buildErr := s.BuildIndex(created)
	if buildErr != nil {
		s.rollbackUpload(created, buildErr)
		return nil, buildErr
	}

	return &model.UploadResult{
		Document:     *created,
		IndexedTerms: terms,
		Duplicated:   false,
	}, nil
}

// rollbackUpload 回滚一次索引构建失败的上传：移除已入库文档及其倒排索引、统计
// 与标签关联，同步索引文档计数并落盘，使文档表与倒排索引恢复到上传前的一致
// 状态。
//
// 该方法不向上返回错误：回滚过程中的落盘失败仅记录日志，上传主流程仍以原始
// 的索引构建错误对外返回，保证调用方能感知到上传并未真正成功。
func (s *Service) rollbackUpload(doc *model.Document, cause error) {
	if doc == nil || doc.ID == "" {
		return
	}
	// DeleteDocument 已包含：移除倒排索引词项、清理统计、递减标签计数、删除文档。
	if err := s.DeleteDocument(doc.ID); err != nil {
		logger.Error("上传回滚：删除文档失败", "doc_id", doc.ID, "err", err.Error())
	}
	// DeleteDocument 的落盘先于此步，且其写入的索引计数尚为回滚前的值，故此处
	// 同步索引文档计数后再次落盘，写入回滚后一致的状态，避免重启后出现文档表
	// 与倒排索引计数错位。
	s.store.SyncIndexDocCount()
	if err := s.store.Save(); err != nil {
		logger.Error("上传回滚：落盘失败", "doc_id", doc.ID, "err", err.Error())
	}
	logger.Error("上传已回滚：索引构建失败", "doc_id", doc.ID, "err", cause.Error())
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
