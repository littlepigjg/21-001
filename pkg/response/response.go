// Package response 提供统一的 HTTP JSON 响应格式与工具函数。
//
// 所有对外接口均使用该包生成响应，保证错误码、消息与数据结构的统一。
package response

import (
	"encoding/json"
	"net/http"
)

// Body 是统一响应体结构。
type Body struct {
	// Code 是业务状态码，0 表示成功，非 0 表示失败。
	Code int `json:"code"`
	// Message 是人类可读的状态描述。
	Message string `json:"message"`
	// Data 是响应携带的业务数据，可为 nil。
	Data interface{} `json:"data,omitempty"`
}

// 业务状态码定义。
const (
	// CodeOK 表示成功。
	CodeOK = 0
	// CodeBadRequest 表示请求参数错误。
	CodeBadRequest = 40000
	// CodeNotFound 表示资源不存在。
	CodeNotFound = 40400
	// CodeConflict 表示资源冲突。
	CodeConflict = 40900
	// CodeTooLarge 表示请求体/文件过大。
	CodeTooLarge = 41300
	// CodeInternal 表示服务内部错误。
	CodeInternal = 50000
)

// write 以给定 HTTP 状态码写出 JSON 响应。
func write(w http.ResponseWriter, status int, body Body) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// Success 写出成功响应（HTTP 200）。
func Success(w http.ResponseWriter, data interface{}) {
	write(w, http.StatusOK, Body{Code: CodeOK, Message: "ok", Data: data})
}

// Created 写出创建成功响应（HTTP 201）。
func Created(w http.ResponseWriter, data interface{}) {
	write(w, http.StatusCreated, Body{Code: CodeOK, Message: "created", Data: data})
}

// NoContent 写出无内容成功响应（HTTP 204）。
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// Error 写出业务错误响应，根据业务码映射 HTTP 状态码。
func Error(w http.ResponseWriter, code int, message string) {
	write(w, httpStatus(code), Body{Code: code, Message: message})
}

// httpStatus 将业务状态码映射为 HTTP 状态码。
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
