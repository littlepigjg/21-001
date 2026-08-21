package handler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"benzhi/internal/config"
	"benzhi/internal/service"
	"benzhi/internal/store"
)

// TestBugContext023_ReuseCanceledContext 验证请求 context 被跨请求复用后，
// 第二次请求是否会被第一次请求已取消的 context 污染。
//
// 期望行为：每次请求都应使用各自独立的 context，第一次请求取消后，
// 第二次全新的请求必须正常返回 200。
func TestBugContext023_ReuseCanceledContext(t *testing.T) {
	cfg := config.Get()
	cfg.Storage.DataDir = t.TempDir()
	cfg.Storage.AutoSave = false

	st, err := store.NewStore(cfg.Storage)
	if err != nil {
		t.Fatalf("构建存储失败: %v", err)
	}
	svc := service.New(st, cfg)
	h := NewHandler(svc)

	// 第一次请求使用可取消的 context，模拟客户端随后主动断开连接。
	ctx1, cancel1 := context.WithCancel(context.Background())
	req1 := httptest.NewRequest(http.MethodGet, "/api/search?q=foobar", nil).WithContext(ctx1)
	rec1 := httptest.NewRecorder()
	h.Search(rec1, req1)

	// 第一次请求结束后，客户端取消该请求的 context。
	cancel1()

	// 第二次请求携带全新的、未取消的 context，本应独立执行。
	req2 := httptest.NewRequest(http.MethodGet, "/api/search?q=foobar", nil)
	rec2 := httptest.NewRecorder()
	h.Search(rec2, req2)

	if rec2.Code != http.StatusOK {
		fmt.Printf("RED（红灯，缺陷未修复）：第二次请求本应正常返回 200，实际返回 %d，body=%s\n", rec2.Code, rec2.Body.String())
		t.Fatalf("第二次请求被第一次请求已取消的 context 污染，期望 200，实际 %d", rec2.Code)
	}
	fmt.Printf("GREEN（绿灯，缺陷已修复）：第二次请求在全新 context 下正常返回 %d\n", rec2.Code)
}
