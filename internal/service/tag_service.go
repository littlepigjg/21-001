package service

import (
	"strings"

	"benzhi/internal/model"
	"benzhi/pkg/util"
)

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

func (s *Service) EnsureTag(name string) (*model.Tag, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, model.ErrInvalidArgument
	}
	if tag, err := s.store.GetTagByName(name); err == nil {
		return tag, nil
	}
	return s.CreateTag(name)
}

func (s *Service) ListTags() ([]*model.Tag, error) {
	return s.store.ListTags()
}

func (s *Service) DeleteTag(id string) error {
	tag, err := s.store.GetTagByID(id)
	if err != nil {
		return err
	}

	if err := s.store.DeleteTag(id); err != nil {
		return err
	}

	docs, _ := s.store.ListDocuments()
	for _, d := range docs {
		if d.HasTag(tag.Name) {
			d.RemoveTag(tag.Name)
			_ = s.store.UpdateDocument(d)
		}
	}
	return nil
}

func (s *Service) TagCount() int {
	tags, err := s.store.ListTags()
	if err != nil {
		return 0
	}
	return len(tags)
}
