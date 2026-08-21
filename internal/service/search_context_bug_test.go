package service

import (
	"context"
	"fmt"
	"os"
	"testing"

	"benzhi/internal/config"
	"benzhi/internal/model"
	"benzhi/internal/store"
)

// TestBugContext001_SearchIgnoresCancellation 验证检索循环是否正确响应 ctx.Err()。
//
// RED（缺陷存在）：传入已取消的 context，Search 仍处理全部候选文档并返回结果。
// GREEN（缺陷修复）：传入已取消的 context，Search 立即返回 context.Canceled 错误。
func TestBugContext001_SearchIgnoresCancellation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "benzhi-bug-ctx-001-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := config.Config{
		Storage: config.StorageConfig{
			DataDir:          tmpDir,
			DocumentsFile:    "docs.json",
			IndexFile:        "idx.json",
			TagsFile:         "tags.json",
			CategoriesFile:   "cats.json",
			StatsFile:        "stats.json",
			AutoSave:         false,
		},
		Search: config.SearchConfig{
			MaxPageSize:      1000,
			DefaultPageSize:  500,
			BM25K1:           1.2,
			BM25B:            0.75,
		},
	}

	st, err := store.NewStore(cfg.Storage)
	if err != nil {
		t.Fatalf("创建存储失败: %v", err)
	}
	svc := New(st, cfg)

	const docCount = 100
	for i := 0; i < docCount; i++ {
		doc := &model.Document{
			ID:      fmt.Sprintf("doc-%04d", i),
			Title:   fmt.Sprintf("文档%d", i),
			Content: fmt.Sprintf("这是关于搜索测试的文档编号%d，包含一些测试内容用于验证上下文取消行为", i),
			Format:  "txt",
		}
		if err := st.CreateDocument(doc); err != nil {
			t.Fatalf("创建文档失败: %v", err)
		}
		st.AddPosting("搜索", doc.ID, []int{0})
		st.AddPosting("测试", doc.ID, []int{2})
	}
	st.SyncIndexDocCount()

	req := &model.SearchRequest{
		Query: "搜索测试",
		Page:  1,
		PageSize: 500,
		SortBy:   "relevance",
	}

	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := svc.Search(cancelledCtx, req)

	if err != nil {
		fmt.Printf("GREEN（绿灯，缺陷已修复）：context 已取消时 Search 正确返回错误: %v\n", err)
		return
	}

	if result != nil && result.Total > 0 {
		fmt.Printf("RED（红灯，缺陷未修复）：context 已取消时 Search 仍处理了全部 %d 条文档并返回 %d 条结果\n", docCount, result.Total)
		t.Fatalf("缺陷未修复：传入已取消的 context 时 Search 不应返回结果，实际返回 %d 条", result.Total)
	}

	fmt.Printf("RED（红灯，缺陷未修复）：context 已取消时 Search 返回了空结果但未返回错误\n")
	t.Fatalf("缺陷未修复：传入已取消的 context 时 Search 应返回 context.Canceled 错误")
}
