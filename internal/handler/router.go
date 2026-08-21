package handler

import (
	"net/http"
)

// NewRouter 构建应用的路由表并包裹中间件。
func NewRouter(h *Handler) http.Handler {
	mux := http.NewServeMux()

	// 健康检查。
	mux.HandleFunc("GET /health", h.Health)
	mux.HandleFunc("GET /ready", h.Ready)

	// 文档相关。
	mux.HandleFunc("GET /api/documents", h.ListDocuments)
	mux.HandleFunc("POST /api/documents", h.CreateDocument)
	mux.HandleFunc("POST /api/documents/upload", h.UploadDocument)
	mux.HandleFunc("GET /api/documents/{id}", h.GetDocument)
	mux.HandleFunc("PUT /api/documents/{id}", h.UpdateDocument)
	mux.HandleFunc("DELETE /api/documents/{id}", h.DeleteDocument)
	mux.HandleFunc("GET /api/documents/{id}/download", h.DownloadDocument)
	mux.HandleFunc("GET /api/documents/{id}/preview", h.PreviewDocument)
	mux.HandleFunc("GET /api/documents/{id}/related", h.RecommendRelated)

	// 检索。
	mux.HandleFunc("GET /api/search", h.Search)
	mux.HandleFunc("GET /api/search/advanced", h.SearchAdvanced)
	mux.HandleFunc("GET /api/suggest", h.Suggest)

	// 标签。
	mux.HandleFunc("GET /api/tags", h.ListTags)
	mux.HandleFunc("POST /api/tags", h.CreateTag)
	mux.HandleFunc("DELETE /api/tags/{id}", h.DeleteTag)

	// 分类。
	mux.HandleFunc("GET /api/categories", h.ListCategories)
	mux.HandleFunc("POST /api/categories", h.CreateCategory)
	mux.HandleFunc("DELETE /api/categories/{id}", h.DeleteCategory)

	// 统计。
	mux.HandleFunc("GET /api/stats/hot", h.HotDocuments)
	mux.HandleFunc("GET /api/stats/{id}", h.DocumentStats)

	// 概览与导入导出。
	mux.HandleFunc("GET /api/metrics", h.Metrics)
	mux.HandleFunc("GET /api/export/json", h.ExportJSON)
	mux.HandleFunc("GET /api/export/csv", h.ExportCSV)
	mux.HandleFunc("POST /api/import", h.ImportJSON)

	// 静态前端页面。
	mux.Handle("/", http.FileServer(http.Dir("./web")))

	return withMiddleware(mux)
}
