package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"benzhi/internal/service"
)

type Handler struct {
	svc *service.Service
}

func NewHandler(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

const maxBodyBytes = 1 << 20

func decodeJSON(r *http.Request, target interface{}) error {
	limited := io.LimitReader(r.Body, maxBodyBytes)
	defer r.Body.Close()

	dec := json.NewDecoder(limited)
	dec.DisallowUnknownFields()
	if err := dec.Decode(target); err != nil {
		return err
	}

	if dec.More() {
		return errors.New("请求体包含多余数据")
	}
	return nil
}

func readAllLimited(r *http.Request, limit int64) ([]byte, error) {
	limited := io.LimitReader(r.Body, limit)
	defer r.Body.Close()
	return io.ReadAll(limited)
}
