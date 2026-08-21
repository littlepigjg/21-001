// Package service 实现系统的业务逻辑层。
//
// 业务逻辑层编排存储层与分词器等基础组件，向上层 handler 提供与具体业务
// 场景强相关的能力，包括：文档生命周期管理、索引构建、全文检索、统计、
// 标签与分类管理、文件上传解析等。
package service

import (
	"benzhi/internal/config"
	"benzhi/internal/store"
	"benzhi/pkg/tokenizer"
)

// Service 是业务逻辑层的核心结构，聚合依赖的基础组件。
type Service struct {
	// store 是底层存储。
	store *store.Store
	// cfg 是应用配置。
	cfg config.Config
	// tokenizer 是文本分词器。
	tokenizer *tokenizer.Tokenizer
}

// New 创建一个 Service 实例。
func New(st *store.Store, cfg config.Config) *Service {
	return &Service{
		store:     st,
		cfg:       cfg,
		tokenizer: tokenizer.New(),
	}
}

// Store 返回底层存储（供 handler 等少数场景访问）。
func (s *Service) Store() *store.Store {
	return s.store
}
