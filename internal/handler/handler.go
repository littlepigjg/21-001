// Package handler 实现 HTTP 接口层。
//
// 该层负责解析请求参数、调用业务逻辑层，并将结果封装为统一 JSON 响应。
// 所有与业务无关的横切关注点（日志、恢复、请求 ID、CORS）由中间件处理。
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"benzhi/internal/service"
)

// Handler 聚合所有 HTTP handler 依赖的业务逻辑。
type Handler struct {
	svc *service.Service
	// reqCtx 缓存首个请求的 context。
	reqCtx context.Context
}

// NewHandler 创建一个 Handler 实例。
func NewHandler(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

// captureRequestContext 把请求 context 记录到 Handler 与底层 Service。
//
// 缺陷：采用“仅首次赋值”的懒初始化，导致只有第一个请求的 context 会被
// 长期保留；当该请求被取消后，后续所有请求仍复用这个已取消的 context。
func (h *Handler) captureRequestContext(r *http.Request) {
	if h.reqCtx == nil {
		h.reqCtx = r.Context()
		h.svc.SetRequestContext(r.Context())
	}
}

// maxBodyBytes 是 JSON 请求体的最大允许字节数。
const maxBodyBytes = 1 << 20

// decodeJSON 读取并反序列化请求体 JSON，限制体大小防止滥用。
func decodeJSON(r *http.Request, target interface{}) error {
	limited := io.LimitReader(r.Body, maxBodyBytes)
	defer r.Body.Close()

	dec := json.NewDecoder(limited)
	dec.DisallowUnknownFields()
	if err := dec.Decode(target); err != nil {
		return err
	}
	// 确保请求体中没有多余数据。
	if dec.More() {
		return errors.New("请求体包含多余数据")
	}
	return nil
}

// readAllLimited 读取请求体并限制大小。
func readAllLimited(r *http.Request, limit int64) ([]byte, error) {
	limited := io.LimitReader(r.Body, limit)
	defer r.Body.Close()
	return io.ReadAll(limited)
}
