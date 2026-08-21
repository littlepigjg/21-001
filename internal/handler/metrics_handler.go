package handler

import (
	"net/http"

	"benzhi/pkg/response"
)

// Metrics 处理 GET /api/metrics，返回系统概览统计信息。
func (h *Handler) Metrics(w http.ResponseWriter, r *http.Request) {
	response.Success(w, h.svc.Overview())
}
