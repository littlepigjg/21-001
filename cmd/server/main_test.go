package main

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"benzhi/internal/config"
	"benzhi/internal/store"
)

// TestGracefulShutdownTimeout 验证“优雅关闭超时未处理导致进程挂起”缺陷。
//
// 缺陷未修复时：store.Save 泄露写操作计数，store.Close 在 wg.Wait() 处永久
// 阻塞，gracefulShutdown 又同步调用 Close 且不设超时，导致整体挂起（RED）。
// 缺陷修复后：Close 能在超时时间内返回，优雅关闭正常完成（GREEN）。
func TestGracefulShutdownTimeout(t *testing.T) {
	cfg := config.StorageConfig{
		DataDir:        t.TempDir(),
		DocumentsFile:  "documents.json",
		IndexFile:      "index.json",
		TagsFile:       "tags.json",
		CategoriesFile: "categories.json",
		StatsFile:      "stats.json",
		AutoSave:       true,
	}

	st, err := store.NewStore(cfg)
	if err != nil {
		t.Fatalf("创建存储失败: %v", err)
	}

	// 触发一次落盘：缺陷存在时，Save 会泄露内部写操作计数。
	if err := st.Save(); err != nil {
		t.Fatalf("首次落盘失败: %v", err)
	}

	srv := &http.Server{Addr: "127.0.0.1:0"}

	done := make(chan error, 1)
	go func() {
		done <- gracefulShutdown(srv, st, 2*time.Second)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("GREEN（绿灯，缺陷已修复）失败：优雅关闭返回错误: %v", err)
		}
		fmt.Println("GREEN（绿灯，缺陷已修复）：优雅关闭在超时时间内正常返回")
	case <-time.After(4 * time.Second):
		t.Fatalf("RED（红灯，缺陷未修复）：优雅关闭超过 4 秒仍未返回，进程挂起")
	}
}
