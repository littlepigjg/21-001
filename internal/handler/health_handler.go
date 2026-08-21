package handler

import (
	"net/http"

	"benzhi/pkg/response"
	"benzhi/pkg/util"
)

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	response.Success(w, map[string]interface{}{
		"status":  "ok",
		"version": util.VersionString(),
	})
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil || h.svc.Store() == nil {
		response.Error(w, response.CodeInternal, "存储未就绪")
		return
	}
	response.Success(w, map[string]interface{}{
		"status": "ready",
	})
}
