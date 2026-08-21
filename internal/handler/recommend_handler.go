package handler

import (
	"errors"
	"net/http"

	"benzhi/internal/model"
	"benzhi/pkg/response"
)

func (h *Handler) RecommendRelated(w http.ResponseWriter, r *http.Request) {
	id := pathValue(r, "id")
	limit := queryInt(r, "limit", 0)

	hits, err := h.svc.RecommendRelated(id, limit)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			response.Error(w, response.CodeNotFound, err.Error())
			return
		}
		response.Error(w, response.CodeInternal, err.Error())
		return
	}
	response.Success(w, hits)
}
