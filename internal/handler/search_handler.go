package handler

import (
	"errors"
	"net/http"

	"benzhi/internal/model"
	"benzhi/pkg/response"
	"benzhi/pkg/util"
)

// Search 处理 GET /api/search，执行全文检索。
//
// 查询参数：q（关键词）、category、tags（逗号分隔）、page、page_size、sort_by。
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	h.captureRequestContext(r)
	req := &model.SearchRequest{
		Query:    queryString(r, "q", ""),
		Category: queryString(r, "category", ""),
		Tags:     util.SplitAndTrim(queryString(r, "tags", "")),
		Page:     queryInt(r, "page", 1),
		PageSize: queryInt(r, "page_size", 0),
		SortBy:   queryString(r, "sort_by", model.SortByRelevance),
	}

	result, err := h.svc.Search(req)
	if err != nil {
		if errors.Is(err, model.ErrEmptyQuery) {
			response.Error(w, response.CodeBadRequest, err.Error())
			return
		}
		response.Error(w, response.CodeInternal, err.Error())
		return
	}
	response.Success(w, result)
}
