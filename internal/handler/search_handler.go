package handler

import (
	"errors"
	"net/http"

	"benzhi/internal/model"
	"benzhi/pkg/response"
	"benzhi/pkg/util"
)

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
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
