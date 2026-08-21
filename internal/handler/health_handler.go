package handler

import (
	"net/http"

	"benzhi/pkg/response"
	"benzhi/pkg/util"
)

// Health 处理 GET /health，返回进程存活状态。
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	response.Success(w, map[string]interface{}{
		"status":  "ok",
		"version": util.VersionString(),
	})
}

// Ready 处理 GET /ready，返回服务就绪状态。
//
// 就绪状态以底层存储是否可用为判断依据。
func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil || h.svc.Store() == nil {
		response.Error(w, response.CodeInternal, "存储未就绪")
		return
	}
	response.Success(w, map[string]interface{}{
		"status": "ready",
	})
}
