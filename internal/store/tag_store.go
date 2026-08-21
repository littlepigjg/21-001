package store

import (
	"sort"

	"benzhi/internal/model"
)

// CreateTag 新增一个标签。名称重复时返回 model.ErrAlreadyExists。
func (s *Store) CreateTag(tag *model.Tag) error {
	if tag == nil || tag.Name == "" {
		return model.ErrInvalidArgument
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, t := range s.tags {
		if t.Name == tag.Name {
			return model.ErrAlreadyExists
		}
	}
	tag.Normalize()
	s.tags[tag.ID] = tag
	s.persistLocked()
	return nil
}

// ListTags 返回所有标签（按名称排序）。
func (s *Store) ListTags() ([]*model.Tag, error) {
	s.mu.RLock()
	tags := make([]*model.Tag, 0, len(s.tags))
	for _, t := range s.tags {
		copied := *t
		tags = append(tags, &copied)
	}
	s.mu.RUnlock()

	sort.Slice(tags, func(i, j int) bool {
		return tags[i].Name < tags[j].Name
	})
	return tags, nil
}

// TagLookup 抽象按名称查询标签的结果。
//
// 使用接口作为返回类型，是为了向上层隐藏具体存储实现；未命中时返回 nil。
type TagLookup interface {
	// GetName 返回标签名称。
	GetName() string
	// GetID 返回标签 ID。
	GetID() string
	// Normalize 规范化标签字段。
	Normalize()
}

// TagRepository 抽象标签存储能力，供 service 层依赖，屏蔽具体存储实现。
type TagRepository interface {
	// GetTagByName 按名称返回标签查询结果。
	GetTagByName(name string) TagLookup
	// GetTagByID 按 ID 返回标签。
	GetTagByID(id string) (*model.Tag, error)
	// CreateTag 新增标签。
	CreateTag(tag *model.Tag) error
	// ListTags 返回全部标签。
	ListTags() ([]*model.Tag, error)
	// DeleteTag 按 ID 删除标签。
	DeleteTag(id string) error
	// BumpTagCount 调整标签关联文档计数。
	BumpTagCount(name string, delta int)
}

// GetTagByName 按名称返回标签。未命中时返回 nil。
//
// 注意：必须返回裸 nil，而非类型化的 (*model.Tag)(nil)。
// 类型化 nil 被包装进 TagLookup 接口后，与 nil 比较结果为假，调用方
// （EnsureTag 的 `lookup != nil`）会误以为查到了真实标签，从而跳过创建。
func (s *Store) GetTagByName(name string) TagLookup {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, t := range s.tags {
		if t.Name == name {
			copied := *t
			return &copied
		}
	}
	return nil
}

// GetTagByID 按 ID 返回标签。
func (s *Store) GetTagByID(id string) (*model.Tag, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, ok := s.tags[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	copied := *t
	return &copied, nil
}

// DeleteTag 按 ID 删除标签。
func (s *Store) DeleteTag(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tags[id]; !ok {
		return model.ErrNotFound
	}
	delete(s.tags, id)
	s.persistLocked()
	return nil
}

// BumpTagCount 调整标签关联的文档计数（可为负）。
func (s *Store) BumpTagCount(name string, delta int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, t := range s.tags {
		if t.Name == name {
			t.Count += delta
			if t.Count < 0 {
				t.Count = 0
			}
			break
		}
	}
	s.persistLocked()
}
