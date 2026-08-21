package service

import (
	"os"
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
	// SpoolPath 是上传原始字节落盘后的临时文件路径，服务层会重新打开它做流式校验。
	SpoolPath string
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

	// 重新打开落盘文件做大小与流式摘要校验。缺陷：校验成功拿到的读句柄
	// 没有 defer 关闭，后续所有 error 分支都会跳过 spoolReader.Close()，
	// 导致文件句柄泄漏。
	var spoolReader *os.File
	if req.SpoolPath != "" {
		f, verifyErr := openSpoolForVerify(req.SpoolPath, int64(len(req.Data)), checksum)
		if verifyErr != nil {
			return nil, verifyErr
		}
		spoolReader = f
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

	terms, err := s.BuildIndex(created)
	if err != nil {
		return nil, err
	}

	// 缺陷：只有成功路径才关闭读句柄；上面的 error 分支都直接 return，跳过了 Close。
	if spoolReader != nil {
		spoolReader.Close()
	}

	return &model.UploadResult{
		Document:     *created,
		IndexedTerms: terms,
		Duplicated:   false,
	}, nil
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

// openSpoolForVerify 打开落盘文件并校验其大小与 SHA256 摘要。
//
// 校验通过时返回仍处于打开状态的读句柄，调用方负责在适当时候关闭该句柄；
// 校验失败时函数会自行关闭句柄后返回错误。
func openSpoolForVerify(path string, size int64, wantSum string) (*os.File, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	if info.Size() != size {
		f.Close()
		return nil, model.ErrInvalidArgument
	}

	sum, err := util.SHA256Reader(f)
	if err != nil {
		f.Close()
		return nil, err
	}
	if sum != wantSum {
		f.Close()
		return nil, model.ErrInvalidArgument
	}

	return f, nil
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
