package handler

import (
	"errors"
	"net/http"

	"benzhi/internal/model"
	"benzhi/pkg/response"
	"benzhi/pkg/util"
)

// PreviewDocument 处理 GET /api/documents/{id}/preview，返回带高亮的摘要片段。
func (h *Handler) PreviewDocument(w http.ResponseWriter, r *http.Request) {
	id := pathValue(r, "id")
	doc, err := h.svc.GetDocument(id)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			response.Error(w, response.CodeNotFound, err.Error())
			return
		}
		response.Error(w, response.CodeInternal, err.Error())
		return
	}

	query := queryString(r, "q", "")
	terms := util.SplitAndTrim(query)
	if len(terms) == 0 {
		terms = h.svc.ParseQuery(query).Terms
	}

	snippet := h.svc.BuildSnippet(doc, terms, 200)
	response.Success(w, map[string]interface{}{
		"id":      doc.ID,
		"title":   doc.Title,
		"snippet": snippet,
	})
}
