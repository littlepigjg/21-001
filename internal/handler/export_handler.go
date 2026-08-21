package handler

import (
	"io"
	"net/http"

	"benzhi/pkg/response"
)

func (h *Handler) ExportJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="documents.json"`)
	if err := h.svc.ExportDocumentsJSON(w); err != nil {
		response.Error(w, response.CodeInternal, err.Error())
		return
	}
}

func (h *Handler) ExportCSV(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="documents.csv"`)
	if err := h.svc.ExportDocumentsCSV(w); err != nil {
		response.Error(w, response.CodeInternal, err.Error())
		return
	}
}

func (h *Handler) ImportJSON(w http.ResponseWriter, r *http.Request) {
	count, err := h.svc.ImportDocumentsJSON(io.LimitReader(r.Body, maxBodyBytes*4))
	if err != nil {
		response.Error(w, response.CodeBadRequest, err.Error())
		return
	}
	response.Success(w, map[string]interface{}{"imported": count})
}
