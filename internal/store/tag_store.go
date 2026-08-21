package store

import (
	"sort"

	"benzhi/internal/model"
)

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
