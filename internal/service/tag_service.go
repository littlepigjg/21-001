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
	// 注意：GetTagByName 未命中时返回 nil。不能只比较接口是否非 nil——
	// 若存储层返回类型化 nil（(*model.Tag)(nil) 包装进接口），接口与 nil
	// 比较为假，会误判标签已存在而跳过创建。此处对断言出的指针再做一次
	// nil 校验，确保任一为空都走创建分支。
	if lookup := s.tagRepo.GetTagByName(name); lookup != nil {
		if tag, ok := lookup.(*model.Tag); ok && tag != nil {
			return tag, nil
		}
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
		// 防御性校验：EnsureTag 理论上不应返回 nil，但若存储层再次出现
		// 类型化 nil 接口之类的回归，这里需避免追加 nil 指针或对不存在的
		// 标签计数，否则会出现“文档挂着标签名但无标签记录”的空洞。
		if tag == nil {
			return nil, model.ErrInvalidArgument
		}
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
