package handler

import (
	"io"
	"net/http"

	"benzhi/pkg/response"
)

// ExportJSON 处理 GET /api/export/json，导出全部文档为 JSON。
func (h *Handler) ExportJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="documents.json"`)
	if err := h.svc.ExportDocumentsJSON(w); err != nil {
		response.Error(w, response.CodeInternal, err.Error())
		return
	}
}

// ExportCSV 处理 GET /api/export/csv，导出全部文档为 CSV。
func (h *Handler) ExportCSV(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="documents.csv"`)
	if err := h.svc.ExportDocumentsCSV(w); err != nil {
		response.Error(w, response.CodeInternal, err.Error())
		return
	}
}

// ImportJSON 处理 POST /api/import，从 JSON 数组导入文档。
func (h *Handler) ImportJSON(w http.ResponseWriter, r *http.Request) {
	count, err := h.svc.ImportDocumentsJSON(io.LimitReader(r.Body, maxBodyBytes*4))
	if err != nil {
		response.Error(w, response.CodeBadRequest, err.Error())
		return
	}
	response.Success(w, map[string]interface{}{"imported": count})
}
