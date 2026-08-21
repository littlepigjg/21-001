package response

import (
	"encoding/json"
	"net/http"
)

type Body struct {
	Code int `json:"code"`

	Message string `json:"message"`

	Data interface{} `json:"data,omitempty"`
}

const (
	CodeOK = 0

	CodeBadRequest = 40000

	CodeNotFound = 40400

	CodeConflict = 40900

	CodeTooLarge = 41300

	CodeInternal = 50000
)

func write(w http.ResponseWriter, status int, body Body) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func Success(w http.ResponseWriter, data interface{}) {
	write(w, http.StatusOK, Body{Code: CodeOK, Message: "ok", Data: data})
}

func Created(w http.ResponseWriter, data interface{}) {
	write(w, http.StatusCreated, Body{Code: CodeOK, Message: "created", Data: data})
}

func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func Error(w http.ResponseWriter, code int, message string) {
	write(w, httpStatus(code), Body{Code: code, Message: message})
}

func httpStatus(code int) int {
	switch {
	case code == CodeBadRequest:
		return http.StatusBadRequest
	case code == CodeNotFound:
		return http.StatusNotFound
	case code == CodeConflict:
		return http.StatusConflict
	case code == CodeTooLarge:
		return http.StatusRequestEntityTooLarge
	case code >= 50000:
		return http.StatusInternalServerError
	default:
		return http.StatusOK
	}
}
