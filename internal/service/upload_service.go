package service

import (
	"os"
	"strings"

	"benzhi/internal/model"
	"benzhi/pkg/util"
)

type UploadRequest struct {
	Filename  string
	Data      []byte
	Title     string
	Category  string
	Tags      []string
	SpoolPath string
}

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

	if spoolReader != nil {
		spoolReader.Close()
	}

	return &model.UploadResult{
		Document:     *created,
		IndexedTerms: terms,
		Duplicated:   false,
	}, nil
}

func (s *Service) isFormatAllowed(format string) bool {
	for _, f := range s.cfg.Upload.AllowedFormats {
		if model.NormalizeFormat(f) == format {
			return true
		}
	}
	return false
}

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

func extractText(format string, data []byte) string {
	switch format {
	case model.FormatPDF:
		return util.ExtractPDFText(data)
	default:
		return string(data)
	}
}
