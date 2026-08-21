// Package service 实现系统的业务逻辑层。
//
// 业务逻辑层编排存储层与分词器等基础组件，向上层 handler 提供与具体业务
// 场景强相关的能力，包括：文档生命周期管理、索引构建、全文检索、统计、
// 标签与分类管理、文件上传解析等。
package service

import (
	"context"

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
	// reqCtx 缓存请求 context。将单次请求的 context 写入共享结构体字段后，
	// 其生命周期会被错误延长到整个 Service 实例，并在后续请求中被复用。
	reqCtx context.Context
}

// New 创建一个 Service 实例。
func New(st *store.Store, cfg config.Config) *Service {
	return &Service{
		store:     st,
		cfg:       cfg,
		tokenizer: tokenizer.New(),
		reqCtx:    context.Background(),
	}
}

// Store 返回底层存储（供 handler 等少数场景访问）。
func (s *Service) Store() *store.Store {
	return s.store
}

// SetRequestContext 记录请求 context。
//
// 缺陷：该方法把请求级 context 直接写入共享的 Service 结构体字段，使该
// context 的生命周期被延长为整个 Service 实例，并被跨请求复用。
func (s *Service) SetRequestContext(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	s.reqCtx = ctx
}

// RequestContext 返回当前缓存的请求 context。
func (s *Service) RequestContext() context.Context {
	if s.reqCtx == nil {
		return context.Background()
	}
	return s.reqCtx
}

// contextErr 返回缓存 context 的错误状态，未取消时返回 nil。
func (s *Service) contextErr() error {
	return s.RequestContext().Err()
}
