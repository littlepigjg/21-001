package handler

import (
	"errors"
	"net/http"

	"benzhi/internal/model"
	"benzhi/pkg/response"
)

type createTagRequest struct {
	Name string `json:"name"`
}

func (h *Handler) ListTags(w http.ResponseWriter, r *http.Request) {
	tags, err := h.svc.ListTags()
	if err != nil {
		response.Error(w, response.CodeInternal, err.Error())
		return
	}
	response.Success(w, tags)
}

func (h *Handler) CreateTag(w http.ResponseWriter, r *http.Request) {
	var req createTagRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, response.CodeBadRequest, "请求体解析失败: "+err.Error())
		return
	}

	tag, err := h.svc.CreateTag(req.Name)
	if err != nil {
		if errors.Is(err, model.ErrAlreadyExists) {
			response.Error(w, response.CodeConflict, err.Error())
			return
		}
		response.Error(w, response.CodeBadRequest, err.Error())
		return
	}
	response.Created(w, tag)
}

func (h *Handler) DeleteTag(w http.ResponseWriter, r *http.Request) {
	id := pathValue(r, "id")
	if err := h.svc.DeleteTag(id); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			response.Error(w, response.CodeNotFound, err.Error())
			return
		}
		response.Error(w, response.CodeInternal, err.Error())
		return
	}
	response.NoContent(w)
}
