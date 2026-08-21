package handler

import (
	"errors"
	"io"
	"net/http"
	"os"

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

	// 将原始上传字节落盘为 spool 临时文件，供服务层做流式摘要校验。
	spool, spoolPath, err := spoolRawUpload(data)
	if err != nil {
		response.Error(w, response.CodeInternal, "落盘上传文件失败: "+err.Error())
		return
	}
	// 缺陷：此处缺少 defer cleanupSpool(spool, spoolPath)。
	// 当 svc.UploadDocument 返回错误时，下面直接 return，spool 句柄与落盘文件都不会被释放。

	result, err := h.svc.UploadDocument(&service.UploadRequest{
		Filename:  header.Filename,
		Data:      data,
		Title:     r.FormValue("title"),
		Category:  r.FormValue("category"),
		Tags:      util.SplitAndTrim(r.FormValue("tags")),
		SpoolPath: spoolPath,
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

	// 缺陷：只有成功路径才清理 spool 文件；错误分支跳过了清理。
	cleanupSpool(spool, spoolPath)

	if result.Duplicated {
		response.Success(w, result)
		return
	}
	response.Created(w, result)
}

// spoolRawUpload 将上传字节写入临时文件并同步到磁盘。
//
// 返回仍处于打开状态的文件句柄以及其路径；调用方负责在适当时候清理该文件。
func spoolRawUpload(data []byte) (*os.File, string, error) {
	f, err := os.CreateTemp("", "benzhi-upload-*.spool")
	if err != nil {
		return nil, "", err
	}

	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(f.Name())
		return nil, "", err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(f.Name())
		return nil, "", err
	}

	return f, f.Name(), nil
}

// cleanupSpool 关闭并删除落盘的上传临时文件。
func cleanupSpool(f *os.File, path string) {
	if f != nil {
		f.Close()
	}
	if path != "" {
		os.Remove(path)
	}
}
