package service

import (
	"fmt"
	"os"
	"testing"

	"benzhi/internal/config"
	"benzhi/internal/model"
	"benzhi/internal/store"
)

// TestBugSlice012_PositionAliasingPollution 验证跨文件缺陷：
// BuildIndex 以引用方式把位置切片交给 store，store 的 addPostingLocked 复用
// 共享缓冲 s.posScratch 存储 Posting.Positions（append 复用底层数组），
// 导致先入索引的文档位置被后续建索引覆盖，进而使 Search 的短语匹配返回错误结果。
func TestBugSlice012_PositionAliasingPollution(t *testing.T) {
	dir, err := os.MkdirTemp("", "benzhi-slice-012-")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(dir)

	cfg := config.Config{
		Storage: config.StorageConfig{
			DataDir:        dir,
			DocumentsFile:  "documents.json",
			IndexFile:      "index.json",
			TagsFile:       "tags.json",
			CategoriesFile: "categories.json",
			StatsFile:      "stats.json",
			AutoSave:       false,
		},
		Search: config.SearchConfig{
			MaxPageSize:      100,
			DefaultPageSize:  10,
			BM25K1:           1.2,
			BM25B:            0.75,
		},
		Upload: config.UploadConfig{
			MaxUploadBytes: 10 * 1024 * 1024,
			AllowedFormats: []string{"txt", "md", "markdown", "pdf"},
		},
	}

	st, err := store.NewStore(cfg.Storage)
	if err != nil {
		t.Fatalf("创建 Store 失败: %v", err)
	}
	svc := New(st, cfg)

	// 上传两篇共享词项 alpha 的文档。
	if _, err := svc.UploadDocument(&UploadRequest{
		Filename: "a.txt",
		Data:     []byte("alpha beta"),
		Title:    "doc-a",
	}); err != nil {
		t.Fatalf("上传 doc-a 失败: %v", err)
	}
	if _, err := svc.UploadDocument(&UploadRequest{
		Filename: "b.txt",
		Data:     []byte("alpha gamma"),
		Title:    "doc-b",
	}); err != nil {
		t.Fatalf("上传 doc-b 失败: %v", err)
	}

	// 打印实际存储的位置，便于观察污染现象。
	dump := func(term string) {
		pl := st.GetPostingList(term)
		for _, p := range pl.Postings {
			fmt.Printf("[debug] term=%q doc=%q positions=%v tf=%d\n", term, p.DocID, p.Positions, p.TermFreq)
		}
	}
	dump("alpha")
	dump("beta")
	dump("gamma")

	res, err := svc.Search(&model.SearchRequest{
		Query:  "alpha beta",
		Phrase: "alpha beta",
	})
	if err != nil {
		t.Fatalf("短语检索失败: %v", err)
	}

	hitDocA := false
	for _, h := range res.Hits {
		if h.Document.Title == "doc-a" {
			hitDocA = true
		}
	}

	if hitDocA {
		fmt.Println("GREEN（绿灯，缺陷已修复）")
		return
	}

	fmt.Println("RED（红灯，缺陷未修复）")
	t.Fatalf("短语 \"alpha beta\" 应命中 doc-a，但实际命中数为 %d；doc-a 的位置被后续建索引污染", res.Total)
}
