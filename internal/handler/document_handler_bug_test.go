package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"benzhi/internal/config"
	"benzhi/internal/service"
	"benzhi/internal/store"
)

// newBugTestRouter 构建一个使用临时数据目录的 Handler，避免污染真实数据。
func newBugTestRouter(t *testing.T) http.Handler {
	t.Helper()

	cfg := config.Get()
	cfg.Storage.DataDir = t.TempDir()
	cfg.Storage.AutoSave = false

	st, err := store.NewStore(cfg.Storage)
	if err != nil {
		t.Fatalf("创建存储失败: %v", err)
	}
	svc := service.New(st, cfg)
	return NewRouter(NewHandler(svc))
}

// TestBugError019_GetDocumentNotFound 验证“请求不存在的文档”时错误处理分支
// 是否正确返回 404，而不是被字符串比较缺陷误导为 500。
//
// 缺陷未修复时应打印 RED（红灯，缺陷未修复）并失败；
// 缺陷修复后应打印 GREEN（绿灯，缺陷已修复）并通过。
func TestBugError019_GetDocumentNotFound(t *testing.T) {
	router := newBugTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/documents/doc-not-exist", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code == http.StatusNotFound {
		t.Logf("GREEN（绿灯，缺陷已修复）: 请求不存在的文档返回 HTTP %d，错误处理分支正确返回 404", rec.Code)
		return
	}
	t.Errorf("RED（红灯，缺陷未修复）: 请求不存在的文档应返回 HTTP 404，实际返回 HTTP %d", rec.Code)
}
