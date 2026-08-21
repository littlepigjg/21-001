package main

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"benzhi/internal/config"
	"benzhi/internal/store"
)

// TestGracefulShutdownTimeout 验证优雅关闭在超时时间内能够正常完成。
//
// 该用例覆盖曾经的两个挂起点：store.Save 泄露写操作计数导致 store.Close 在
// wg.Wait() 处永久阻塞，以及 gracefulShutdown 同步调用 Close 且不设超时。
// 修复后 Save 归还计数，Close 不再阻塞，gracefulShutdown 走带超时的
// CloseWithContext，优雅关闭在超时时间内正常返回。
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

	// 触发一次落盘：调用方需自行确认 Save 内部计数已正确归还，否则后续
	// Close 会因 wg.Wait() 永久阻塞。
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
