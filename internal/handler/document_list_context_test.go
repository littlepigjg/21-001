package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"benzhi/internal/config"
	"benzhi/internal/model"
	"benzhi/internal/service"
	"benzhi/internal/store"
)

// TestBugContext022_ListDocumentsCancelPropagation 验证文档列表请求取消后，
// 取消信号是否被正确传播到下游存储遍历：
//   - 缺陷未修复时：客户端已取消，但服务端仍继续遍历并返回 200（RED）。
//   - 缺陷已修复时：客户端取消被传播，服务端中断遍历并返回错误（GREEN）。
func TestBugContext022_ListDocumentsCancelPropagation(t *testing.T) {
	cfg := config.Get()
	cfg.Storage.DataDir = t.TempDir()
	cfg.Storage.AutoSave = false

	st, err := store.NewStore(cfg.Storage)
	if err != nil {
		t.Fatalf("创建存储失败: %v", err)
	}

	for i := 0; i < 200; i++ {
		doc := &model.Document{
			ID:         fmt.Sprintf("doc-%04d", i),
			Title:      fmt.Sprintf("文档 %d", i),
			Content:    "这是一段用于测试取消传播的正文内容",
			UploadTime: int64(i),
		}
		if err := st.CreateDocument(doc); err != nil {
			t.Fatalf("插入文档失败: %v", err)
		}
	}

	svc := service.New(st, cfg)
	h := NewHandler(svc)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req := httptest.NewRequest(http.MethodGet, "/api/documents?page=1&page_size=10", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	h.ListDocuments(rec, req)

	resp := rec.Result()
	defer resp.Body.Close()

	var body map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&body)

	message, _ := body["message"].(string)
	code := resp.StatusCode

	// 修复后：取消被传播，handler 应返回 5xx，且消息包含 "context canceled"。
	fixed := code >= 400 && strings.Contains(message, "context canceled")

	if fixed {
		fmt.Printf("GREEN（绿灯，缺陷已修复）：取消被正确传播，服务端中断遍历，HTTP=%d，message=%q\n", code, message)
		return
	}

	fmt.Printf("RED（红灯，缺陷未修复）：请求已取消但服务端仍继续遍历并返回成功，HTTP=%d，message=%q\n", code, message)
	t.Fatalf("期望取消后返回错误，实际 HTTP=%d，message=%q", code, message)
}
