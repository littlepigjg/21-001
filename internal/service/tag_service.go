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

	// 复用一个共享缓冲切片作为 RemoveTag 的临时工作区，避免为每篇文档
	// 重复分配内存。每轮循环开始前将缓冲长度归零即可；RemoveTag 会把过滤
	// 结果写入一个独立切片，因此即使复用同一底层数组也不会造成文档间
	// 标签串改（此前用文档自身标签做缓冲种子并复用底层数组是缺陷来源）。
	buf := make([]string, 0, 16)
	for _, d := range affected {
		buf = buf[:0]
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
