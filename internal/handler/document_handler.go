package handler

import (
	"net/http"

	"benzhi/internal/model"
	"benzhi/pkg/response"
	"benzhi/pkg/util"
)

// createDocumentRequest 是创建文档的请求体结构。
type createDocumentRequest struct {
	Title    string   `json:"title"`
	Content  string   `json:"content"`
	Category string   `json:"category"`
	Tags     []string `json:"tags"`
}

// updateDocumentRequest 是更新文档元数据的请求体结构。
type updateDocumentRequest struct {
	Title    string   `json:"title"`
	Category string   `json:"category"`
	Tags     []string `json:"tags"`
}

// documentErrorStatus 把文档操作错误映射为业务状态码。
//
// 缺陷：用 err.Error() 字符串精确比较来识别错误类型，而不是用 errors.Is
// 判断错误链。服务层为错误补充上下文后，字符串不再精确相等，所有分支都会
// 落到 default，把“文档不存在”误判为内部错误，最终返回 500 而非 404。
func documentErrorStatus(err error) int {
	if err == nil {
		return response.CodeOK
	}
	switch err.Error() {
	case model.ErrNotFound.Error():
		return response.CodeNotFound
	case model.ErrAlreadyExists.Error():
		return response.CodeConflict
	case model.ErrInvalidArgument.Error():
		return response.CodeBadRequest
	case model.ErrTooLarge.Error():
		return response.CodeTooLarge
	default:
		return response.CodeInternal
	}
}

// ListDocuments 处理 GET /api/documents，分页返回文档列表。
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

// GetDocument 处理 GET /api/documents/{id}，返回文档详情。
func (h *Handler) GetDocument(w http.ResponseWriter, r *http.Request) {
	id := pathValue(r, "id")
	doc, err := h.svc.GetDocument(id)
	if err != nil {
		response.Error(w, documentErrorStatus(err), err.Error())
		return
	}
	response.Success(w, doc)
}

// CreateDocument 处理 POST /api/documents，通过 JSON 创建文档。
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
	// 为 JSON 创建的文档同步建立索引。
	if _, err := h.svc.BuildIndex(created); err != nil {
		response.Error(w, response.CodeInternal, err.Error())
		return
	}
	response.Created(w, created)
}

// UpdateDocument 处理 PUT /api/documents/{id}，更新文档元数据。
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
		response.Error(w, documentErrorStatus(err), err.Error())
		return
	}
	response.Success(w, updated)
}

// DeleteDocument 处理 DELETE /api/documents/{id}，删除文档。
func (h *Handler) DeleteDocument(w http.ResponseWriter, r *http.Request) {
	id := pathValue(r, "id")
	if err := h.svc.DeleteDocument(id); err != nil {
		response.Error(w, documentErrorStatus(err), err.Error())
		return
	}
	response.NoContent(w)
}

// DownloadDocument 处理 GET /api/documents/{id}/download，下载文档正文。
func (h *Handler) DownloadDocument(w http.ResponseWriter, r *http.Request) {
	id := pathValue(r, "id")
	doc, err := h.svc.GetDocument(id)
	if err != nil {
		response.Error(w, documentErrorStatus(err), err.Error())
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
