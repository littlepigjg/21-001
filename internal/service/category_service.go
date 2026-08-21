package service

import (
	"strings"

	"benzhi/internal/model"
	"benzhi/pkg/util"
)

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

func (s *Service) ListCategories() ([]*model.Category, error) {
	return s.store.ListCategories()
}

func (s *Service) DeleteCategory(id string) error {
	cat, err := s.store.GetCategoryByID(id)
	if err != nil {
		return err
	}

	if err := s.store.DeleteCategory(id); err != nil {
		return err
	}

	docs, _ := s.store.ListDocuments()
	for _, d := range docs {
		if d.Category == cat.Name {
			d.Category = ""
			_ = s.store.UpdateDocument(d)
		}
	}
	return nil
}

func (s *Service) CategoryCount() int {
	cats, err := s.store.ListCategories()
	if err != nil {
		return 0
	}
	return len(cats)
}
