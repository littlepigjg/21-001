package handler

import (
	"errors"
	"net/http"

	"benzhi/internal/model"
	"benzhi/pkg/response"
)

// HotDocuments 处理 GET /api/stats/hot，返回按热度排序的文档。
func (h *Handler) HotDocuments(w http.ResponseWriter, r *http.Request) {
	limit := queryInt(r, "limit", 0)
	hits, err := h.svc.PopularDocuments(limit)
	if err != nil {
		response.Error(w, response.CodeInternal, err.Error())
		return
	}
	response.Success(w, hits)
}

// DocumentStats 处理 GET /api/stats/{id}，返回单个文档的统计信息。
func (h *Handler) DocumentStats(w http.ResponseWriter, r *http.Request) {
	id := pathValue(r, "id")
	stats, err := h.svc.GetDocumentStats(id)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			response.Error(w, response.CodeNotFound, err.Error())
			return
		}
		response.Error(w, response.CodeInternal, err.Error())
		return
	}
	response.Success(w, stats)
}
