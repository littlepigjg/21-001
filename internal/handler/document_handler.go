package handler

import (
	"errors"
	"net/http"

	"benzhi/internal/model"
	"benzhi/pkg/response"
	"benzhi/pkg/util"
)

type createDocumentRequest struct {
	Title    string   `json:"title"`
	Content  string   `json:"content"`
	Category string   `json:"category"`
	Tags     []string `json:"tags"`
}

type updateDocumentRequest struct {
	Title    string   `json:"title"`
	Category string   `json:"category"`
	Tags     []string `json:"tags"`
}

func (h *Handler) ListDocuments(w http.ResponseWriter, r *http.Request) {
	page := queryInt(r, "page", 1)
	pageSize := queryInt(r, "page_size", 0)

	docs, total, err := h.svc.ListDocuments(page, pageSize)
	if err != nil {
		response.Error(w, response.CodeInternal, err.Error())
		return
	}
	response.Success(w, map[string]interface{}{
		"total": total,
		"items": docs,
	})
}

func (h *Handler) GetDocument(w http.ResponseWriter, r *http.Request) {
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
	response.Success(w, doc)
}

func (h *Handler) CreateDocument(w http.ResponseWriter, r *http.Request) {
	var req createDocumentRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, response.CodeBadRequest, "请求体解析失败: "+err.Error())
		return
	}

	doc := &model.Document{
		Title:    req.Title,
		Content:  req.Content,
		Category: req.Category,
		Tags:     req.Tags,
		Format:   model.FormatTXT,
	}
	created, err := h.svc.CreateDocument(doc)
	if err != nil {
		response.Error(w, response.CodeBadRequest, err.Error())
		return
	}

	if _, err := h.svc.BuildIndex(created); err != nil {
		response.Error(w, response.CodeInternal, err.Error())
		return
	}
	response.Created(w, created)
}

func (h *Handler) UpdateDocument(w http.ResponseWriter, r *http.Request) {
	id := pathValue(r, "id")
	var req updateDocumentRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, response.CodeBadRequest, "请求体解析失败: "+err.Error())
		return
	}

	updated, err := h.svc.UpdateDocument(id, &model.Document{
		Title:    req.Title,
		Category: req.Category,
		Tags:     req.Tags,
	})
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			response.Error(w, response.CodeNotFound, err.Error())
			return
		}
		response.Error(w, response.CodeBadRequest, err.Error())
		return
	}
	response.Success(w, updated)
}

func (h *Handler) DeleteDocument(w http.ResponseWriter, r *http.Request) {
	id := pathValue(r, "id")
	if err := h.svc.DeleteDocument(id); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			response.Error(w, response.CodeNotFound, err.Error())
			return
		}
		response.Error(w, response.CodeInternal, err.Error())
		return
	}
	response.NoContent(w)
}

func (h *Handler) DownloadDocument(w http.ResponseWriter, r *http.Request) {
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

	if _, err := h.svc.IncrementDownload(id); err != nil {
		response.Error(w, response.CodeInternal, err.Error())
		return
	}

	filename := util.SanitizeFilename(doc.Title) + ".txt"
	w.Header().Set("Content-Type", util.MIMETypeByExtension(filename))
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	_, _ = w.Write([]byte(doc.Content))
}
