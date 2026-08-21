package service

import (
	"strings"

	"benzhi/internal/model"
	"benzhi/pkg/util"
)

// CreateCategory 创建一个新分类。
func (s *Service) CreateCategory(name, description string) (*model.Category, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, model.ErrInvalidArgument
	}

	cat := &model.Category{
		ID:          util.NewIDWithPrefix("cat-"),
		Name:        name,
		Description: strings.TrimSpace(description),
		CreateTime:  util.Now(),
	}
	if err := s.store.CreateCategory(cat); err != nil {
		return nil, err
	}
	return cat, nil
}

// ListCategories 返回全部分类。
func (s *Service) ListCategories() ([]*model.Category, error) {
	return s.store.ListCategories()
}

// DeleteCategory 删除分类。该分类下的文档将被置为空分类。
func (s *Service) DeleteCategory(id string) error {
	cat, err := s.store.GetCategoryByID(id)
	if err != nil {
		return err
	}

	if err := s.store.DeleteCategory(id); err != nil {
		return err
	}

	// 将属于该分类的文档分类字段清空。
	docs, _ := s.store.ListDocuments()
	for _, d := range docs {
		if d.Category == cat.Name {
			d.Category = ""
			_ = s.store.UpdateDocument(d)
		}
	}
	return nil
}

// CategoryCount 返回分类总数。
func (s *Service) CategoryCount() int {
	cats, err := s.store.ListCategories()
	if err != nil {
		return 0
	}
	return len(cats)
}
