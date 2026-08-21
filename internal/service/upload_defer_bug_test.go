package service

import (
	"os"
	"path/filepath"
	"runtime/debug"
	"testing"

	"benzhi/internal/config"
	"benzhi/internal/store"
)

// countOpenFDs 返回当前进程打开的文件描述符数量（依赖 Linux /proc 文件系统）。
func countOpenFDs(t *testing.T) int {
	t.Helper()
	entries, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		t.Fatalf("读取 /proc/self/fd 失败: %v", err)
	}
	return len(entries)
}

// TestBugDefer028_UploadFailedFileLeaksHandle 验证缺陷：
// 重复上传一个“解析失败”的文件（正文为空白）时，service 层的错误分支跳过
// spool 文件句柄释放，导致文件描述符持续泄漏。
//
// 缺陷未修复时：fd 数量随失败次数线性增长，判定为 RED 并使测试失败。
// 缺陷修复后：fd 数量保持稳定，判定为 GREEN 并通过。
func TestBugDefer028_UploadFailedFileLeaksHandle(t *testing.T) {
	cfg := config.Config{
		Upload: config.UploadConfig{
			MaxUploadBytes: 10 * 1024 * 1024,
			AllowedFormats: []string{"txt", "md", "markdown", "pdf"},
		},
	}
	st, err := store.NewStore(config.StorageConfig{DataDir: t.TempDir()})
	if err != nil {
		t.Fatalf("创建 store 失败: %v", err)
	}
	svc := New(st, cfg)

	// 构造一个“正文为空白”的落盘 spool 文件，其内容与 req.Data 一致，
	// 保证流式摘要校验通过，随后稳定进入 content 为空的错误分支。
	spoolDir := t.TempDir()
	spoolPath := filepath.Join(spoolDir, "upload.spool")
	payload := []byte("   \n\t ")
	if err := os.WriteFile(spoolPath, payload, 0o644); err != nil {
		t.Fatalf("写入 spool 文件失败: %v", err)
	}

	// 临时关闭自动 GC，避免 os.File 的 finalizer 在未显式 Close 时回收句柄，
	// 从而让“是否关闭”真正反映在 fd 计数上，保证判定确定性。
	oldGC := debug.SetGCPercent(-1)
	defer debug.SetGCPercent(oldGC)

	before := countOpenFDs(t)

	const iterations = 50
	for i := 0; i < iterations; i++ {
		_, err := svc.UploadDocument(&UploadRequest{
			Filename:  "leak.txt",
			Data:      payload,
			SpoolPath: spoolPath,
		})
		if err == nil {
			t.Fatalf("第 %d 次上传应返回错误，实际无错误", i)
		}
	}

	after := countOpenFDs(t)
	leaked := after - before
	t.Logf("重复上传失败 %d 次后，打开的文件描述符：before=%d, after=%d, 泄漏=%d", iterations, before, after, leaked)

	const threshold = 40
	if leaked >= threshold {
		t.Fatalf("RED（红灯，缺陷未修复）：错误分支未释放 spool 文件句柄，重复上传失败文件导致文件描述符泄漏 %d 个", leaked)
	}
	t.Logf("GREEN（绿灯，缺陷已修复）：错误分支正确释放 spool 文件句柄，未检测到文件描述符泄漏")
}
