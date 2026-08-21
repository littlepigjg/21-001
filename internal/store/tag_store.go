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

// GetTagByName 按名称返回标签。
func (s *Store) GetTagByName(name string) (*model.Tag, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, t := range s.tags {
		if t.Name == name {
			copied := *t
			return &copied, nil
		}
	}
	return nil, model.ErrNotFound
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

// BumpTagCount 原子地调整标签关联的文档计数（可为负）。
//
// 读-改-写完整地发生在写锁内：多个并发调用各自基于最新值自增并写回，
// 不会相互覆盖。这是维护 Count 的唯一正确入口，业务层不应再绕过它
// 在锁外修改标签计数，否则会丢失更新。
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

// GetTagRefByName 按名称返回标签的副本。
//
// 返回的是拷贝而非内部共享指针，调用方在锁外修改返回值不会影响 Store 内部
// 状态，从而避免标签关联计数（Count）的丢失更新。
func (s *Store) GetTagRefByName(name string) (*model.Tag, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, t := range s.tags {
		if t.Name == name {
			copied := *t
			return &copied, nil
		}
	}
	return nil, model.ErrNotFound
}

// SetTagCount 将指定名称标签的关联文档计数覆盖为 count。
//
// 该方法只在确需「整体覆盖」计数时使用（如重建索引等批量场景）。并发上传
// 等需要基于当前值自增的场景必须使用 BumpTagCount，在写锁内原子地完成
// 读-改-写，否则多个 goroutine 会基于同一个旧值计算后相互覆盖。
func (s *Store) SetTagCount(name string, count int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, t := range s.tags {
		if t.Name == name {
			t.Count = count
			if t.Count < 0 {
				t.Count = 0
			}
			break
		}
	}
	s.persistLocked()
}
