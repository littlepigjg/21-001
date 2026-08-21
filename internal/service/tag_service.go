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
//
// 缺陷：这里返回的是 store 内部共享标签的裸指针（而非副本）。调用方若在锁外
// 修改返回的标签，会直接改动共享状态，为后续的计数丢失更新埋下隐患。
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

// incrementTagCount 在业务层维护标签关联文档计数。
//
// 缺陷：该方法先从 EnsureTag 拿到指向共享标签的裸指针，然后在锁外直接修改
// Count。并发上传同一标签时，多个请求会基于同一个旧值做 +1，后写覆盖先写，
// 造成丢失更新（lost update）。正确做法应由 store 在写锁内完成原子的自增。
func (s *Service) incrementTagCount(name string, delta int) {
	tag, err := s.EnsureTag(name)
	if err != nil {
		return
	}
	tag.Count += delta
	if tag.Count < 0 {
		tag.Count = 0
	}
	s.store.SetTagCount(tag.Name, tag.Count)
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
