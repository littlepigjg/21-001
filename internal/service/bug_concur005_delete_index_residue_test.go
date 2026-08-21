package service

import (
	"fmt"
	"testing"

	"benzhi/internal/config"
	"benzhi/internal/model"
	"benzhi/internal/store"
)

// TestBugConcur005_DeleteIndexResidue 验证“文档删除与索引移除非原子导致残留”缺陷。
//
// 场景：删除文档的同时存在可检索到的共享词项，删除后检索仍返回已删除文档。
// 未修复时本用例判定为 RED（红灯，缺陷未修复）；修复后判定为 GREEN（绿灯，缺陷已修复）。
func TestBugConcur005_DeleteIndexResidue(t *testing.T) {
	storageCfg := config.StorageConfig{
		DataDir:        t.TempDir(),
		DocumentsFile:  "documents.json",
		IndexFile:      "index.json",
		TagsFile:       "tags.json",
		CategoriesFile: "categories.json",
		StatsFile:      "stats.json",
		AutoSave:       false,
	}
	st, err := store.NewStore(storageCfg)
	if err != nil {
		t.Fatalf("创建存储失败: %v", err)
	}

	cfg := config.Config{
		Storage: storageCfg,
		Search: config.SearchConfig{
			DefaultPageSize: 10,
			MaxPageSize:     100,
			BM25K1:          1.2,
			BM25B:           0.75,
		},
	}
	svc := New(st, cfg)

	doc1, err := svc.CreateDocument(&model.Document{
		Title:   "doc1",
		Content: "apple banana",
	})
	if err != nil {
		t.Fatalf("创建文档1失败: %v", err)
	}
	doc2, err := svc.CreateDocument(&model.Document{
		Title:   "doc2",
		Content: "banana cherry",
	})
	if err != nil {
		t.Fatalf("创建文档2失败: %v", err)
	}

	if _, err := svc.BuildIndex(doc1); err != nil {
		t.Fatalf("构建文档1索引失败: %v", err)
	}
	if _, err := svc.BuildIndex(doc2); err != nil {
		t.Fatalf("构建文档2索引失败: %v", err)
	}

	if err := svc.DeleteDocument(doc1.ID); err != nil {
		t.Fatalf("删除文档1失败: %v", err)
	}

	res, err := svc.Search(&model.SearchRequest{
		Query:    "banana",
		Page:     1,
		PageSize: 100,
	})
	if err != nil {
		t.Fatalf("检索失败: %v", err)
	}

	// 判定1：检索结果中不应再出现已删除的 doc1。
	searchResidue := false
	for _, hit := range res.Hits {
		if hit.Document.ID == doc1.ID {
			searchResidue = true
		}
	}

	// 判定2：倒排索引中不应再残留 doc1。
	indexResidue := false
	pl := st.GetPostingList("banana")
	for _, p := range pl.Postings {
		if p.DocID == doc1.ID {
			indexResidue = true
		}
	}

	if searchResidue || indexResidue {
		fmt.Printf("【判定】RED（红灯，缺陷未修复） searchResidue=%v indexResidue=%v 检索Total=%d\n",
			searchResidue, indexResidue, res.Total)
		t.Fatalf("文档删除与索引移除非原子导致残留：doc1=%s 删除后仍残留（searchResidue=%v, indexResidue=%v）",
			doc1.ID, searchResidue, indexResidue)
	}

	fmt.Printf("【判定】GREEN（绿灯，缺陷已修复） 检索Total=%d，已删除文档不再返回\n", res.Total)
}
