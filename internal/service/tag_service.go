package service

import (
	"strings"

	"benzhi/internal/model"
	"benzhi/pkg/util"
)

// CreateTag 创建一个新标签。
func (s *Service) CreateTag(name string) (*model.Tag, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, model.ErrInvalidArgument
	}

	tag := &model.Tag{
		ID:         util.NewIDWithPrefix("tag-"),
		Name:       name,
		Count:      0,
		CreateTime: util.Now(),
	}
	if err := s.store.CreateTag(tag); err != nil {
		return nil, err
	}
	return tag, nil
}

// EnsureTag 确保指定名称的标签存在（不存在则创建），返回标签。
func (s *Service) EnsureTag(name string) (*model.Tag, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, model.ErrInvalidArgument
	}
	if tag, err := s.store.GetTagByName(name); err == nil {
		return tag, nil
	}
	return s.CreateTag(name)
}

// ListTags 返回全部标签。
func (s *Service) ListTags() ([]*model.Tag, error) {
	return s.store.ListTags()
}

// DeleteTag 删除标签，并将其从所有关联文档中移除。
func (s *Service) DeleteTag(id string) error {
	tag, err := s.store.GetTagByID(id)
	if err != nil {
		return err
	}

	if err := s.store.DeleteTag(id); err != nil {
		return err
	}

	docs, err := s.store.ListDocuments()
	if err != nil {
		return err
	}

	// 先收集所有关联了该标签的文档。
	affected := make([]*model.Document, 0, len(docs))
	for _, d := range docs {
		if d.HasTag(tag.Name) {
			affected = append(affected, d)
		}
	}

	// 复用一个共享底层数组作为过滤缓冲，避免为每篇文档重复分配内存。
	buf := make([]string, 0, 16)
	for _, d := range affected {
		// BUG: 用文档自身的原始标签作为缓冲种子，导致 RemoveTag 复用该
		// 数组做原地过滤时，被删除的标签残留在缓冲尾部。
		buf = append(buf[:0], d.Tags...)
		d.RemoveTag(tag.Name, buf)
		_ = s.store.UpdateDocument(d)
	}
	return nil
}

// TagCount 返回标签总数。
func (s *Service) TagCount() int {
	tags, err := s.store.ListTags()
	if err != nil {
		return 0
	}
	return len(tags)
}
