package store

import (
	"fmt"
	"path/filepath"
	"sync"

	"benzhi/internal/config"
	"benzhi/internal/model"
	"benzhi/pkg/util"
)

type Store struct {
	mu sync.RWMutex

	cfg config.StorageConfig

	dataDir string

	documents map[string]*model.Document

	index *model.InvertedIndex

	tags map[string]*model.Tag

	categories map[string]*model.Category

	stats map[string]*model.DocumentStats
}

func NewStore(cfg config.StorageConfig) (*Store, error) {
	if err := util.EnsureDir(cfg.DataDir); err != nil {
		return nil, fmt.Errorf("创建数据目录失败: %w", err)
	}

	s := &Store{
		cfg:        cfg,
		dataDir:    cfg.DataDir,
		documents:  make(map[string]*model.Document),
		index:      model.NewInvertedIndex(),
		tags:       make(map[string]*model.Tag),
		categories: make(map[string]*model.Category),
		stats:      make(map[string]*model.DocumentStats),
	}
	return s, nil
}

func (s *Store) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var docs map[string]*model.Document
	if ok, err := util.LoadJSON(s.path(s.cfg.DocumentsFile), &docs); err != nil {
		return err
	} else if ok && docs != nil {
		s.documents = docs
	}

	var idx *model.InvertedIndex
	if ok, err := util.LoadJSON(s.path(s.cfg.IndexFile), &idx); err != nil {
		return err
	} else if ok && idx != nil {
		if idx.Terms == nil {
			idx.Terms = make(map[string]model.PostingList)
		}
		s.index = idx
	}

	var tags map[string]*model.Tag
	if ok, err := util.LoadJSON(s.path(s.cfg.TagsFile), &tags); err != nil {
		return err
	} else if ok && tags != nil {
		s.tags = tags
	}

	var cats map[string]*model.Category
	if ok, err := util.LoadJSON(s.path(s.cfg.CategoriesFile), &cats); err != nil {
		return err
	} else if ok && cats != nil {
		s.categories = cats
	}

	var stats map[string]*model.DocumentStats
	if ok, err := util.LoadJSON(s.path(s.cfg.StatsFile), &stats); err != nil {
		return err
	} else if ok && stats != nil {
		s.stats = stats
	}

	return nil
}

func (s *Store) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.saveLocked()
}

func (s *Store) saveLocked() error {
	if err := util.SaveJSON(s.path(s.cfg.DocumentsFile), s.documents); err != nil {
		return err
	}
	if err := util.SaveJSON(s.path(s.cfg.IndexFile), s.index); err != nil {
		return err
	}
	if err := util.SaveJSON(s.path(s.cfg.TagsFile), s.tags); err != nil {
		return err
	}
	if err := util.SaveJSON(s.path(s.cfg.CategoriesFile), s.categories); err != nil {
		return err
	}
	if err := util.SaveJSON(s.path(s.cfg.StatsFile), s.stats); err != nil {
		return err
	}
	return nil
}

func (s *Store) Close() error {
	return s.Save()
}

func (s *Store) path(name string) string {
	return filepath.Join(s.dataDir, name)
}
