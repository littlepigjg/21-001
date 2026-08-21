package handler

import (
	"errors"
	"net/http"

	"benzhi/internal/model"
	"benzhi/pkg/response"
)

// createCategoryRequest 是创建分类的请求体。
type createCategoryRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ListCategories 处理 GET /api/categories，返回全部分类。
func (h *Handler) ListCategories(w http.ResponseWriter, r *http.Request) {
	cats, err := h.svc.ListCategories()
	if err != nil {
		response.Error(w, response.CodeInternal, err.Error())
		return
	}
	response.Success(w, cats)
}

// CreateCategory 处理 POST /api/categories，创建分类。
func (h *Handler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var req createCategoryRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, response.CodeBadRequest, "请求体解析失败: "+err.Error())
		return
	}

	cat, err := h.svc.CreateCategory(req.Name, req.Description)
	if err != nil {
		if errors.Is(err, model.ErrAlreadyExists) {
			response.Error(w, response.CodeConflict, err.Error())
			return
		}
		response.Error(w, response.CodeBadRequest, err.Error())
		return
	}
	response.Created(w, cat)
}

// DeleteCategory 处理 DELETE /api/categories/{id}，删除分类。
func (h *Handler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	id := pathValue(r, "id")
	if err := h.svc.DeleteCategory(id); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			response.Error(w, response.CodeNotFound, err.Error())
			return
		}
		response.Error(w, response.CodeInternal, err.Error())
		return
	}
	response.NoContent(w)
}
