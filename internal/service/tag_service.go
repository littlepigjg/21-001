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

// EnsureTag 确保指定名称的标签存在（不存在则创建），返回标签的副本。
//
// 返回的标签是拷贝，调用方在锁外修改它不会影响 Store 内部状态，避免标签
// 关联计数（Count）在并发上传时发生丢失更新。
func (s *Service) EnsureTag(name string) (*model.Tag, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, model.ErrInvalidArgument
	}
	if tag, err := s.store.GetTagRefByName(name); err == nil {
		return tag, nil
	}
	return s.CreateTag(name)
}

// incrementTagCount 原子地维护标签关联文档计数。
//
// 计数的读-改-写由 store 在写锁内完成（BumpTagCount），业务层不持有也不
// 修改共享标签指针。并发上传同一标签时，多个 goroutine 各自基于最新值
// 自增并写回，Count 不会相互覆盖。
func (s *Service) incrementTagCount(name string, delta int) {
	// 确保标签存在；计数自增在 store 写锁内原子完成，避免丢失更新。
	if _, err := s.EnsureTag(name); err != nil {
		return
	}
	s.store.BumpTagCount(name, delta)
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

	// 从所有文档中移除该标签。
	docs, _ := s.store.ListDocuments()
	for _, d := range docs {
		if d.HasTag(tag.Name) {
			d.RemoveTag(tag.Name)
			_ = s.store.UpdateDocument(d)
		}
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
