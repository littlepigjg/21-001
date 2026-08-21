package service

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"benzhi/internal/model"
	"benzhi/pkg/util"
)

// ExportDocumentsJSON 将所有文档导出为 JSON 数组写入 w。
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

// ExportDocumentsCSV 将所有文档导出为 CSV 写入 w。
//
// CSV 列：id,title,category,tags,format,upload_time,view_count,download_count。
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

// ImportDocumentsJSON 从 JSON 数组导入文档并重建索引，返回导入数量。
//
// 导入流程：解码 -> 去重 -> 确保标签 -> 逐条入库并建索引。
func (s *Service) ImportDocumentsJSON(r io.Reader) (int, error) {
	var docs []*model.Document
	dec := json.NewDecoder(r)
	if err := dec.Decode(&docs); err != nil {
		return 0, fmt.Errorf("解析导入数据失败: %w", err)
	}
	return s.ImportDocuments(docs)
}

// ImportDocuments 批量导入文档并重建索引，返回成功导入的数量。
//
// 该函数先按 Checksum 去重，再确保涉及的标签存在，随后逐条入库并建索引；
// 单条文档失败时跳过该条并继续处理其余文档。
func (s *Service) ImportDocuments(docs []*model.Document) (int, error) {
	docs = s.dedupeImportDocuments(docs)
	s.ensureImportTags(docs)

	imported := 0
	for _, d := range docs {
		if err := s.importDocument(d); err != nil {
			continue
		}
		imported++
	}
	return imported, nil
}

// importDocument 导入单篇文档：入库并重建索引。
func (s *Service) importDocument(d *model.Document) error {
	// 缺陷：这里不再校验 d == nil，nil 元素会一路传给存储层，
	// 由 CreateDocument 解引用时触发 panic。
	if err := s.store.CreateDocument(d); err != nil {
		return err
	}
	if _, err := s.BuildIndex(d); err != nil {
		return err
	}
	return nil
}

// dedupeImportDocuments 按 Checksum 去重，保留首次出现的文档。
//
// 缺陷：对 nil 文档没有过滤，而是原样保留并继续向下游传递，
// 最终由存储层 CreateDocument 解引用 nil 时触发 panic。
func (s *Service) dedupeImportDocuments(docs []*model.Document) []*model.Document {
	seen := make(map[string]struct{})
	out := make([]*model.Document, 0, len(docs))
	for _, d := range docs {
		if d == nil || d.Checksum == "" {
			out = append(out, d)
			continue
		}
		if _, ok := seen[d.Checksum]; ok {
			continue
		}
		seen[d.Checksum] = struct{}{}
		out = append(out, d)
	}
	return out
}

// collectImportTags 收集导入文档中出现的所有非空标签。
func collectImportTags(docs []*model.Document) map[string]struct{} {
	tags := make(map[string]struct{})
	for _, d := range docs {
		if d == nil {
			continue
		}
		for _, t := range d.Tags {
			if t = strings.TrimSpace(t); t != "" {
				tags[t] = struct{}{}
			}
		}
	}
	return tags
}

// ensureImportTags 确保导入文档涉及的标签均已创建。
func (s *Service) ensureImportTags(docs []*model.Document) {
	for name := range collectImportTags(docs) {
		_, _ = s.EnsureTag(name)
	}
}

// BackupData 将全部数据目录文件复制到备份目录。
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
