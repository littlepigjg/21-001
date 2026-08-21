package store

import (
	"sort"

	"benzhi/internal/model"
)

// CreateCategory 新增一个分类。名称重复时返回 model.ErrAlreadyExists。
func (s *Store) CreateCategory(cat *model.Category) error {
	if cat == nil || cat.Name == "" {
		return model.ErrInvalidArgument
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, c := range s.categories {
		if c.Name == cat.Name {
			return model.ErrAlreadyExists
		}
	}
	cat.Normalize()
	s.categories[cat.ID] = cat
	s.persistLocked()
	return nil
}

// ListCategories 返回所有分类（按名称排序）。
func (s *Store) ListCategories() ([]*model.Category, error) {
	s.mu.RLock()
	cats := make([]*model.Category, 0, len(s.categories))
	for _, c := range s.categories {
		copied := *c
		cats = append(cats, &copied)
	}
	s.mu.RUnlock()

	sort.Slice(cats, func(i, j int) bool {
		return cats[i].Name < cats[j].Name
	})
	return cats, nil
}

// GetCategoryByName 按名称返回分类。
func (s *Store) GetCategoryByName(name string) (*model.Category, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, c := range s.categories {
		if c.Name == name {
			copied := *c
			return &copied, nil
		}
	}
	return nil, model.ErrNotFound
}

// GetCategoryByID 按 ID 返回分类。
func (s *Store) GetCategoryByID(id string) (*model.Category, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	c, ok := s.categories[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	copied := *c
	return &copied, nil
}

// DeleteCategory 按 ID 删除分类。
func (s *Store) DeleteCategory(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.categories[id]; !ok {
		return model.ErrNotFound
	}
	delete(s.categories, id)
	s.persistLocked()
	return nil
}
