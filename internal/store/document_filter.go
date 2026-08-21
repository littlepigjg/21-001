package store

import (
	"sort"
	"strings"

	"benzhi/internal/model"
)

// ListByCategory 返回指定分类下的所有文档（按上传时间倒序）。
func (s *Store) ListByCategory(category string) ([]*model.Document, error) {
	s.mu.RLock()
	var docs []*model.Document
	for _, d := range s.documents {
		if d.Category == category {
			copied := *d
			copied.Tags = append([]string(nil), d.Tags...)
			docs = append(docs, &copied)
		}
	}
	s.mu.RUnlock()

	sort.Slice(docs, func(i, j int) bool {
		return docs[i].UploadTime > docs[j].UploadTime
	})
	return docs, nil
}

// ListByTag 返回包含指定标签的所有文档（按上传时间倒序）。
func (s *Store) ListByTag(tag string) ([]*model.Document, error) {
	s.mu.RLock()
	var docs []*model.Document
	for _, d := range s.documents {
		if d.HasTag(tag) {
			copied := *d
			copied.Tags = append([]string(nil), d.Tags...)
			docs = append(docs, &copied)
		}
	}
	s.mu.RUnlock()

	sort.Slice(docs, func(i, j int) bool {
		return docs[i].UploadTime > docs[j].UploadTime
	})
	return docs, nil
}

// ListByTitleContains 返回标题包含指定子串的文档列表（不区分大小写）。
func (s *Store) ListByTitleContains(substr string) ([]*model.Document, error) {
	substr = strings.ToLower(substr)
	s.mu.RLock()
	var docs []*model.Document
	for _, d := range s.documents {
		if strings.Contains(strings.ToLower(d.Title), substr) {
			copied := *d
			copied.Tags = append([]string(nil), d.Tags...)
			docs = append(docs, &copied)
		}
	}
	s.mu.RUnlock()

	sort.Slice(docs, func(i, j int) bool {
		return docs[i].UploadTime > docs[j].UploadTime
	})
	return docs, nil
}

// ListTagsWithCounts 返回标签及其关联文档数量（动态计算）。
func (s *Store) ListTagsWithCounts() map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	counts := make(map[string]int)
	for _, d := range s.documents {
		for _, t := range d.Tags {
			counts[t]++
		}
	}
	return counts
}
