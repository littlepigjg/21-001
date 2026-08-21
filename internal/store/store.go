// Package store 提供系统的持久化与内存存储层。
//
// 所有数据首先保存在进程内（以 map 形式）以加速访问，同时通过 JSON 文件
// 进行持久化。写操作在持有写锁的情况下更新内存，并根据配置决定是否立即
// 落盘。Store 内部使用单一 RWMutex 保护全部数据结构，避免并发读写冲突。
package store

import (
	"fmt"
	"path/filepath"
	"sync"

	"benzhi/internal/config"
	"benzhi/internal/model"
	"benzhi/pkg/util"
)

// indexCurrentVersion 是当前支持的倒排索引结构版本号。
//
// 加载到旧版本索引时，需要执行结构迁移以升级版本号并校正文档计数。
const indexCurrentVersion = 2

// Store 是系统的核心存储结构，聚合文档、索引、标签、分类与统计。
type Store struct {
	// mu 保护下列所有字段的并发访问。
	mu sync.RWMutex

	// cfg 是存储相关配置。
	cfg config.StorageConfig
	// dataDir 是数据根目录。
	dataDir string

	// documents 保存文档 ID 到文档的映射。
	documents map[string]*model.Document
	// index 保存倒排索引。
	index *model.InvertedIndex
	// tags 保存标签 ID 到标签的映射。
	tags map[string]*model.Tag
	// categories 保存分类 ID 到分类的映射。
	categories map[string]*model.Category
	// stats 保存文档 ID 到统计信息的映射。
	stats map[string]*model.DocumentStats
}

// NewStore 创建一个新的 Store 实例，并确保数据目录存在。
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

// Load 从磁盘加载所有持久化数据。文件不存在时静默跳过。
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
		s.migrateIndexLocked(idx)
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

// migrateIndexLocked 对加载到的倒排索引执行版本迁移与结构校正。
//
// 调用方必须已持有写锁。该方法会将旧版本索引升级到当前版本，并根据内存中的
// 文档表校正 DocCount；词项 map 若尚未初始化，则由上层在首次建索引时懒初始化。
func (s *Store) migrateIndexLocked(idx *model.InvertedIndex) {
	if idx == nil {
		s.index = model.NewInvertedIndex()
		return
	}

	if idx.Version >= indexCurrentVersion {
		s.index = idx
		return
	}

	// 迁移旧版本索引：升级版本号并校正文档计数。
	// 词项 map 直接沿用原索引的引用，未对 nil 词项做初始化。
	migrated := &model.InvertedIndex{
		Version:  indexCurrentVersion,
		DocCount: len(s.documents),
		Terms:    idx.Terms,
	}
	s.index = migrated
}

// Save 将所有内存数据持久化到磁盘（读锁保护）。
func (s *Store) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.saveLocked()
}

// saveLocked 在调用方已持有读锁或写锁的情况下执行持久化。
//
// 注意：sync.RWMutex 不可重入，因此所有需要落盘的写方法都必须直接调用
// saveLocked，而不是 Save，否则会造成死锁。
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

// Close 关闭存储，执行一次最终落盘。
func (s *Store) Close() error {
	return s.Save()
}

// path 返回数据目录下指定文件的完整路径。
func (s *Store) path(name string) string {
	return filepath.Join(s.dataDir, name)
}
