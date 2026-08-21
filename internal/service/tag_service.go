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
	if err := s.tagRepo.CreateTag(tag); err != nil {
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
	if lookup := s.tagRepo.GetTagByName(name); lookup != nil {
		return lookup.(*model.Tag), nil
	}
	return s.CreateTag(name)
}

// ListTags 返回全部标签。
func (s *Service) ListTags() ([]*model.Tag, error) {
	return s.tagRepo.ListTags()
}

// DeleteTag 删除标签，并将其从所有关联文档中移除。
func (s *Service) DeleteTag(id string) error {
	tag, err := s.tagRepo.GetTagByID(id)
	if err != nil {
		return err
	}

	if err := s.tagRepo.DeleteTag(id); err != nil {
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
	tags, err := s.tagRepo.ListTags()
	if err != nil {
		return 0
	}
	return len(tags)
}

// AssociateTags 批量规范化并确保标签存在，递增关联计数，返回关联的标签列表。
//
// 该方法被文档上传流程调用，负责将文档请求中的标签名落地为真实标签记录，
// 并同步维护标签的文档计数。
func (s *Service) AssociateTags(names []string) ([]*model.Tag, error) {
	normalized := s.normalizeTagNames(names)
	tags := make([]*model.Tag, 0, len(normalized))
	for _, name := range normalized {
		tag, err := s.EnsureTag(name)
		if err != nil {
			return nil, err
		}
		// 缺陷：EnsureTag 因 nil 接口缺陷会对新标签返回 nil 指针 + nil error，
		// 这里未校验 tag 是否为 nil 就直接追加并计数，导致新标签未被真正创建。
		tags = append(tags, tag)
		s.tagRepo.BumpTagCount(name, 1)
	}
	return tags, nil
}

// normalizeTagNames 对标签名做去空格、去重并过滤空值。
func (s *Service) normalizeTagNames(names []string) []string {
	seen := make(map[string]bool, len(names))
	out := make([]string, 0, len(names))
	for _, raw := range names {
		name := strings.TrimSpace(raw)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	return out
}
