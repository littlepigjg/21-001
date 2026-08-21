package handler

import (
	"net/http"
)

func NewRouter(h *Handler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", h.Health)
	mux.HandleFunc("GET /ready", h.Ready)

	mux.HandleFunc("GET /api/documents", h.ListDocuments)
	mux.HandleFunc("POST /api/documents", h.CreateDocument)
	mux.HandleFunc("POST /api/documents/upload", h.UploadDocument)
	mux.HandleFunc("GET /api/documents/{id}", h.GetDocument)
	mux.HandleFunc("PUT /api/documents/{id}", h.UpdateDocument)
	mux.HandleFunc("DELETE /api/documents/{id}", h.DeleteDocument)
	mux.HandleFunc("GET /api/documents/{id}/download", h.DownloadDocument)
	mux.HandleFunc("GET /api/documents/{id}/preview", h.PreviewDocument)
	mux.HandleFunc("GET /api/documents/{id}/related", h.RecommendRelated)

	mux.HandleFunc("GET /api/search", h.Search)
	mux.HandleFunc("GET /api/search/advanced", h.SearchAdvanced)
	mux.HandleFunc("GET /api/suggest", h.Suggest)

	mux.HandleFunc("GET /api/tags", h.ListTags)
	mux.HandleFunc("POST /api/tags", h.CreateTag)
	mux.HandleFunc("DELETE /api/tags/{id}", h.DeleteTag)

	mux.HandleFunc("GET /api/categories", h.ListCategories)
	mux.HandleFunc("POST /api/categories", h.CreateCategory)
	mux.HandleFunc("DELETE /api/categories/{id}", h.DeleteCategory)

	mux.HandleFunc("GET /api/stats/hot", h.HotDocuments)
	mux.HandleFunc("GET /api/stats/{id}", h.DocumentStats)

	mux.HandleFunc("GET /api/metrics", h.Metrics)
	mux.HandleFunc("GET /api/export/json", h.ExportJSON)
	mux.HandleFunc("GET /api/export/csv", h.ExportCSV)
	mux.HandleFunc("POST /api/import", h.ImportJSON)

	mux.Handle("/", http.FileServer(http.Dir("./web")))

	return withMiddleware(mux)
}
