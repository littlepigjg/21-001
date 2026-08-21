package handler

import (
	"net/http"

	"benzhi/internal/model"
	"benzhi/pkg/response"
)

// SearchAdvanced 处理 GET /api/search/advanced，支持高级查询语法。
//
// 查询参数 q 支持 "引号短语"、category:值、tag:值 等语法。
func (h *Handler) SearchAdvanced(w http.ResponseWriter, r *http.Request) {
	raw := queryString(r, "q", "")
	pq := h.svc.ParseQuery(raw)

	req := &model.SearchRequest{
		Query:    pq.CombineQuery(),
		Category: pq.Category,
		Tags:     pq.Tags,
		Page:     queryInt(r, "page", 1),
		PageSize: queryInt(r, "page_size", 0),
		SortBy:   queryString(r, "sort_by", model.SortByRelevance),
	}

	result, err := h.svc.Search(req)
	if err != nil {
		response.Error(w, response.CodeBadRequest, err.Error())
		return
	}
	response.Success(w, result)
}
