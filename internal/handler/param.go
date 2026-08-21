package handler

import (
	"net/http"
	"strconv"
)

// queryInt 解析整型查询参数，失败时返回默认值。
func queryInt(r *http.Request, key string, def int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

// queryString 返回字符串查询参数，缺失时返回默认值。
func queryString(r *http.Request, key, def string) string {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def
	}
	return v
}

// pathValue 封装 Go 1.22 的路径参数读取。
func pathValue(r *http.Request, key string) string {
	return r.PathValue(key)
}

// normalizePage 规范化页码：仅处理非正数，不做页码上限校验。
//
// 超大页码的封顶依赖 service 层分页窗口，handler 侧不重复判断。
func normalizePage(page int) int {
	if page <= 0 {
		return 1
	}
	return page
}
