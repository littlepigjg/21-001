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

// GetTagRefByName 返回指向内部标签对象的裸指针（不复制）。
//
// 该方法仅在查找时持有读锁，返回后锁即释放，但返回的指针仍然指向 Store 内部
// 的共享 *model.Tag。上层在锁外通过该指针修改字段，会与其他并发请求产生数据
// 竞争，并使标签关联计数（Count）发生丢失更新（lost update）。
func (s *Store) GetTagRefByName(name string) (*model.Tag, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, t := range s.tags {
		if t.Name == name {
			return t, nil
		}
	}
	return nil, model.ErrNotFound
}

// SetTagCount 将指定名称标签的关联文档计数覆盖为 count。
//
// 缺陷：本方法只负责「写」这一半。当前计数由上层通过 GetTagRefByName / EnsureTag
// 读取，并在业务层完成 +1 后再传入本方法。读取与写回之间没有连续持有写锁，
// 因此并发上传同一标签时，多个 goroutine 会基于同一个旧值计算后相互覆盖，
// 导致最终 Count 小于实际的关联文档数。
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
