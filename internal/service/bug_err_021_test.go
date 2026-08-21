package service

import (
	"fmt"
	"testing"

	"benzhi/internal/config"
	"benzhi/internal/model"
	"benzhi/internal/store"
)

// newBugTestService 构造一个使用临时目录、开箱即用的 Service 与 Store 实例。
func newBugTestService(t *testing.T) (*Service, *store.Store) {
	t.Helper()

	cfg := config.Config{
		Storage: config.StorageConfig{
			DataDir:        t.TempDir(),
			DocumentsFile:  "documents.json",
			IndexFile:      "index.json",
			TagsFile:       "tags.json",
			CategoriesFile: "categories.json",
			StatsFile:      "stats.json",
			AutoSave:       true,
		},
		Search: config.SearchConfig{
			MaxPageSize:     100,
			DefaultPageSize: 10,
			BM25K1:          1.2,
			BM25B:           0.75,
		},
		Upload: config.UploadConfig{
			MaxUploadBytes: 10 * 1024 * 1024,
			AllowedFormats: []string{"txt", "md", "markdown", "pdf"},
		},
	}

	st, err := store.NewStore(cfg.Storage)
	if err != nil {
		t.Fatalf("创建存储失败: %v", err)
	}
	return New(st, cfg), st
}

// TestBugErr021_UploadIgnoresIndexBuildError 验证：索引构建/落盘失败时，
// 上传主流程不应忽略该错误，否则会出现“上传成功但检索不到”的文档与索引
// 状态不一致问题。
func TestBugErr021_UploadIgnoresIndexBuildError(t *testing.T) {
	svc, st := newBugTestService(t)

	// 注入索引落盘故障：模拟磁盘写入失败，使 BuildIndex 的 FlushIndex 返回错误。
	st.SetFailIndexFlush(true)

	req := &UploadRequest{
		Filename: "hello.txt",
		Data:     []byte("hello world"),
	}

	result, err := svc.UploadDocument(req)
	if err != nil {
		// 修复后的正确行为：上传在索引构建失败时应返回错误，
		// 不会留下“已入库但无索引”的脏文档。
		fmt.Println("GREEN（绿灯，缺陷已修复）：上传在索引构建失败时正确返回错误")
		return
	}

	// 走到这里说明上传返回了成功（缺陷路径）：索引错误被忽略，
	// 文档已入库，但索引可能已经缺失。
	search, serr := svc.Search(&model.SearchRequest{Query: "hello"})
	if serr != nil {
		t.Fatalf("检索失败: %v", serr)
	}

	found := false
	for _, hit := range search.Hits {
		if hit.Document.ID == result.Document.ID {
			found = true
			break
		}
	}

	if found {
		fmt.Println("GREEN（绿灯，缺陷已修复）：上传成功且文档可被检索到")
		return
	}

	// 打印索引与文档表的不一致信息，便于定位缺陷。
	for _, issue := range st.IndexConsistency() {
		fmt.Println("不一致诊断：", issue)
	}

	fmt.Println("RED（红灯，缺陷未修复）：上传返回成功，但该文档检索不到")
	t.Fatalf("文档 %s 上传成功但索引缺失，检索不到", result.Document.ID)
}
