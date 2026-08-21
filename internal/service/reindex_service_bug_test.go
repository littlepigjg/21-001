package service

import (
	"fmt"
	"testing"

	"benzhi/internal/config"
	"benzhi/internal/model"
	"benzhi/internal/store"
)

// TestRebuildIndexTermCount 验证重建索引返回的词项计数是否正确。
//
// 缺陷未修复时返回的词项计数会错误地等于文档数，本用例会打印 RED 并失败；
// 缺陷修复后返回正确的唯一词项数，本用例会打印 GREEN 并通过。
func TestRebuildIndexTermCount(t *testing.T) {
	st, err := store.NewStore(config.StorageConfig{
		DataDir:        t.TempDir(),
		DocumentsFile:  "documents.json",
		IndexFile:      "index.json",
		TagsFile:       "tags.json",
		CategoriesFile: "categories.json",
		StatsFile:      "stats.json",
	})
	if err != nil {
		t.Fatalf("创建存储失败: %v", err)
	}

	svc := New(st, config.Config{})

	docs := []*model.Document{
		{ID: "doc-1", Content: "alpha beta"},
		{ID: "doc-2", Content: "beta gamma"},
	}
	for _, d := range docs {
		if err := st.CreateDocument(d); err != nil {
			t.Fatalf("创建文档失败: %v", err)
		}
	}

	indexed, termCount, err := svc.RebuildIndex()
	if err != nil {
		t.Fatalf("重建索引失败: %v", err)
	}

	const (
		wantIndexed   = 2
		wantTermCount = 3
	)

	if indexed == wantIndexed && termCount == wantTermCount {
		fmt.Printf("GREEN（绿灯，缺陷已修复）：indexed=%d, termCount=%d\n", indexed, termCount)
		return
	}

	fmt.Printf("RED（红灯，缺陷未修复）：indexed=%d(期望 %d), termCount=%d(期望 %d)\n",
		indexed, wantIndexed, termCount, wantTermCount)
	t.Errorf("重建索引返回的计数错误: indexed=%d(期望 %d), termCount=%d(期望 %d)",
		indexed, wantIndexed, termCount, wantTermCount)
}
