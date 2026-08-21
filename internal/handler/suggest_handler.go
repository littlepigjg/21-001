package handler

import (
	"net/http"

	"benzhi/pkg/response"
)

// Suggest 处理 GET /api/suggest，返回检索联想词项。
func (h *Handler) Suggest(w http.ResponseWriter, r *http.Request) {
	prefix := queryString(r, "prefix", "")
	limit := queryInt(r, "limit", 10)
	terms := h.svc.Suggest(prefix, limit)
	response.Success(w, terms)
}
