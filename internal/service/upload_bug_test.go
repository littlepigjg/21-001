package service

import (
	"os"
	"path/filepath"
	"testing"

	"benzhi/internal/config"
	"benzhi/internal/store"
)

// TestBugError018_UploadErrorShadowing 验证上传流程中“内层 err 被 := 遮蔽导致错误被忽略”的缺陷。
//
// 缺陷未修复时：文档/索引持久化失败被 := 遮蔽，UploadDocument 仍返回成功（RED）。
// 缺陷修复后：持久化失败会正确向上返回错误（GREEN）。
func TestBugError018_UploadErrorShadowing(t *testing.T) {
	t.Run("文档持久化失败被遮蔽", func(t *testing.T) {
		svc := newFailingUploadService(t, "documents.json")
		_, err := svc.UploadDocument(&UploadRequest{
			Filename: "hello.txt",
			Data:     []byte("hello world this is a test document"),
		})
		if err == nil {
			t.Errorf("RED（红灯，缺陷未修复）：文档持久化失败仍返回成功，错误被 := 遮蔽")
			return
		}
		t.Logf("GREEN（绿灯，缺陷已修复）：文档持久化失败正确返回错误")
	})

	t.Run("索引持久化失败被遮蔽", func(t *testing.T) {
		svc := newFailingUploadService(t, "index.json")
		_, err := svc.UploadDocument(&UploadRequest{
			Filename: "hello.txt",
			Data:     []byte("hello world this is a test document"),
		})
		if err == nil {
			t.Errorf("RED（红灯，缺陷未修复）：索引持久化失败仍返回成功，错误被 := 遮蔽")
			return
		}
		t.Logf("GREEN（绿灯，缺陷已修复）：索引持久化失败正确返回错误")
	})
}

// newFailingUploadService 构造一个指定文件持久化会失败的上传服务。
//
// 通过把目标文件名占位为目录，使 util.SaveJSON 的临时文件重命名失败，
// 从而在不依赖外部环境变量的前提下稳定触发持久化错误。
func newFailingUploadService(t *testing.T, failingFile string) *Service {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, failingFile), 0o755); err != nil {
		t.Fatal(err)
	}

	cfg := config.Get()
	cfg.Storage = config.StorageConfig{
		DataDir:        dir,
		DocumentsFile:  "documents.json",
		IndexFile:      "index.json",
		TagsFile:       "tags.json",
		CategoriesFile: "categories.json",
		StatsFile:      "stats.json",
		AutoSave:       true,
	}

	st, err := store.NewStore(cfg.Storage)
	if err != nil {
		t.Fatal(err)
	}
	return New(st, cfg)
}
