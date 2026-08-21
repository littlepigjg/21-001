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

const maxMultipartMemory = 32 << 20

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

	spool, spoolPath, err := spoolRawUpload(data)
	if err != nil {
		response.Error(w, response.CodeInternal, "落盘上传文件失败: "+err.Error())
		return
	}

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

	cleanupSpool(spool, spoolPath)

	if result.Duplicated {
		response.Success(w, result)
		return
	}
	response.Created(w, result)
}

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

func cleanupSpool(f *os.File, path string) {
	if f != nil {
		f.Close()
	}
	if path != "" {
		os.Remove(path)
	}
}
