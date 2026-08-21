package service

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"benzhi/internal/model"
	"benzhi/pkg/util"
)

func (s *Service) ExportDocumentsJSON(w io.Writer) error {
	docs, err := s.store.ListDocuments()
	if err != nil {
		return err
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(docs); err != nil {
		return fmt.Errorf("导出 JSON 失败: %w", err)
	}
	return nil
}

func (s *Service) ExportDocumentsCSV(w io.Writer) error {
	docs, err := s.store.ListDocuments()
	if err != nil {
		return err
	}

	headers := []string{"id", "title", "category", "tags", "format", "upload_time", "view_count", "download_count"}
	rows := make([][]string, 0, len(docs))
	for _, d := range docs {
		st := s.store.GetStats(d.ID)
		rows = append(rows, []string{
			d.ID,
			d.Title,
			d.Category,
			strings.Join(d.Tags, "|"),
			d.Format,
			fmt.Sprintf("%d", d.UploadTime),
			fmt.Sprintf("%d", st.ViewCount),
			fmt.Sprintf("%d", st.DownloadCount),
		})
	}
	if _, err := util.WriteCSV(w, headers, rows); err != nil {
		return fmt.Errorf("导出 CSV 失败: %w", err)
	}
	return nil
}

func (s *Service) ImportDocumentsJSON(r io.Reader) (int, error) {
	var docs []*model.Document
	dec := json.NewDecoder(r)
	if err := dec.Decode(&docs); err != nil {
		return 0, fmt.Errorf("解析导入数据失败: %w", err)
	}

	imported := 0
	for _, d := range docs {
		if d == nil || d.Title == "" || d.Content == "" {
			continue
		}

		d.ID = util.NewIDWithPrefix("doc-")
		d.Normalize()
		if err := s.store.CreateDocument(d); err != nil {
			continue
		}
		if _, err := s.BuildIndex(d); err != nil {
			continue
		}
		imported++
	}
	return imported, nil
}

func (s *Service) BackupData(backupDir string) (int, error) {
	files, err := util.ListFiles(s.cfg.Storage.DataDir)
	if err != nil {
		return 0, err
	}

	copied := 0
	for _, f := range files {
		dst := backupDir + "/" + strings.TrimPrefix(f, s.cfg.Storage.DataDir+"/")
		if err := util.CopyFile(f, dst); err != nil {
			return copied, err
		}
		copied++
	}
	return copied, nil
}
