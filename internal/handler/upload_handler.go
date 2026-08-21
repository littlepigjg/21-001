package handler

import (
	"errors"
	"io"
	"net/http"

	"benzhi/internal/model"
	"benzhi/internal/service"
	"benzhi/pkg/response"
	"benzhi/pkg/util"
)

// maxMultipartMemory 是 multipart 解析的内存缓冲上限。
const maxMultipartMemory = 32 << 20

// UploadDocument 处理 POST /api/documents/upload，接收文件上传。
//
// 表单字段：file（必填）、title、category、tags（逗号分隔）。
func (h *Handler) UploadDocument(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxMultipartMemory); err != nil {
		response.Error(w, response.CodeBadRequest, "解析上传表单失败: "+err.Error())
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		response.Error(w, response.CodeBadRequest, "缺少上传文件: "+err.Error())
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		response.Error(w, response.CodeInternal, "读取上传文件失败: "+err.Error())
		return
	}

	result, err := h.svc.UploadDocument(&service.UploadRequest{
		Filename: header.Filename,
		Data:     data,
		Title:    r.FormValue("title"),
		Category: r.FormValue("category"),
		Tags:     util.SplitAndTrim(r.FormValue("tags")),
	})
	if err != nil {
		switch {
		case errors.Is(err, model.ErrTooLarge):
			response.Error(w, response.CodeTooLarge, err.Error())
		case errors.Is(err, model.ErrUnsupportedFormat), errors.Is(err, model.ErrInvalidArgument):
			response.Error(w, response.CodeBadRequest, err.Error())
		default:
			response.Error(w, response.CodeInternal, err.Error())
		}
		return
	}

	if result.Duplicated {
		response.Success(w, result)
		return
	}
	response.Created(w, result)
}
