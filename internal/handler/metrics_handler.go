package handler

import (
	"net/http"

	"benzhi/pkg/response"
)

func (h *Handler) Metrics(w http.ResponseWriter, r *http.Request) {
	response.Success(w, h.svc.Overview())
}
