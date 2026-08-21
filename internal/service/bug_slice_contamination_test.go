package service

import (
	"benzhi/internal/config"
	"benzhi/internal/model"
	"benzhi/internal/store"
	"os"
	"testing"
)

// TestBugSlice016_PostingListSubsliceContamination 验证倒排列表子切片写回
// 导致的底层数组污染缺陷。
//
// 缺陷描述：
// RemoveDocumentFromIndex 使用 pl.Postings[:0] 子切片构建过滤后的
// 倒排列表，复用原切片的底层数组。当多个词项的 Posting 切片共享同一
// 底层数组时，过滤一个词项的记录会覆盖底层数组中其他词项的数据。
// 同时 RemoveIndex 不重建索引，导致被污染的数据持续影响后续检索。
//
// 预期行为：
// - 缺陷存在时（RED）：删除文档后，搜索重叠词项会返回已删除文档的记录
// - 缺陷修复后（GREEN）：删除文档后，搜索重叠词项只返回未删除的文档
func TestBugSlice016_PostingListSubsliceContamination(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bug-slice-016-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := config.Config{
		Storage: config.StorageConfig{
			DataDir:        tmpDir,
			DocumentsFile:  "docs.json",
			IndexFile:      "idx.json",
			TagsFile:       "tags.json",
			CategoriesFile: "cats.json",
			StatsFile:      "stats.json",
			AutoSave:       false,
		},
		Search: config.SearchConfig{
			MaxPageSize:     100,
			DefaultPageSize: 10,
			BM25K1:          1.2,
			BM25B:           0.75,
		},
	}

	st, err := store.NewStore(cfg.Storage)
	if err != nil {
		t.Fatalf("创建 Store 失败: %v", err)
	}
	svc := New(st, cfg)

	// 创建两个有大量重叠词项的文档。
	doc1 := &model.Document{
		ID:      "doc-1",
		Title:   "文档一",
		Content: "倒排索引是全文检索的核心数据结构 用于快速定位包含特定词项的文档列表 通过词项到文档的映射实现高效检索",
		Tags:    []string{"search", "index"},
	}
	doc2 := &model.Document{
		ID:      "doc-2",
		Title:   "文档二",
		Content: "倒排索引是全文检索的核心数据结构 用于快速定位包含特定词项的文档列表 通过词项到文档的映射实现高效检索 性能优化是关键",
		Tags:    []string{"search", "optimization"},
	}

	// 手动创建文档记录（跳过 CreateDocument 的校验，直接写入 store）。
	if err := st.CreateDocument(doc1); err != nil {
		t.Fatalf("创建文档1失败: %v", err)
	}
	if err := st.CreateDocument(doc2); err != nil {
		t.Fatalf("创建文档2失败: %v", err)
	}

	// 手动构建倒排索引，确保两个文档的 Posting 切片共享同一底层数组。
	// 通过直接调用 AddPosting 来模拟这个场景：先为 doc1 添加词项，
	// 再为 doc2 添加相同词项，由于 Go 的 append 机制，它们的 Posting
	// 切片会共享同一底层数组。
	indexDoc(t, svc, doc1)
	indexDoc(t, svc, doc2)

	// 删除文档2。
	svc.RemoveIndex(doc2.ID)

	// 搜索"倒排索引"（两个文档都包含的词项）。
	searchReq := &model.SearchRequest{
		Query:    "倒排索引",
		Page:     1,
		PageSize: 100,
	}
	result, err := svc.Search(searchReq)
	if err != nil {
		t.Fatalf("搜索失败: %v", err)
	}

	// 检查搜索结果。
	// 删除文档2后，搜索"倒排索引"应该只返回文档1。
	// 如果缺陷存在（子切片写回污染底层数组），文档2的记录可能仍然
	// 出现在搜索结果中。
	foundDoc1 := false
	foundDoc2 := false
	for _, hit := range result.Hits {
		if hit.Document.ID == doc1.ID {
			foundDoc1 = true
		}
		if hit.Document.ID == doc2.ID {
			foundDoc2 = true
		}
	}

	// 判定结果。
	if !foundDoc1 {
		t.Log("========== RED（红灯，缺陷未修复） ==========")
		t.Logf("  期望：搜索结果包含文档1（doc-1），实际：未找到文档1")
		t.Logf("  搜索返回 %d 条结果，Total=%d", result.Total, result.Total)
		t.Error("RED: 搜索结果应包含文档1（doc-1），但未找到")
		return
	}

	if foundDoc2 {
		t.Log("========== RED（红灯，缺陷未修复） ==========")
		t.Logf("  期望：搜索结果不包含已删除的文档2（doc-2），实际：文档2仍出现在搜索结果中")
		t.Logf("  搜索返回 %d 条结果，Total=%d", result.Total, result.Total)
		for i, hit := range result.Hits {
			t.Logf("  结果[%d]: docID=%s, title=%s", i, hit.Document.ID, hit.Document.Title)
		}
		t.Error("RED: 搜索结果不应包含已删除的文档2（doc-2），但仍然出现")
		return
	}

	// 额外验证：搜索 doc2 独有的词项"性能优化"，应返回0条结果。
	searchReq2 := &model.SearchRequest{
		Query:    "性能优化",
		Page:     1,
		PageSize: 100,
	}
	result2, err := svc.Search(searchReq2)
	if err != nil {
		t.Fatalf("搜索失败: %v", err)
	}
	if result2.Total > 0 {
		t.Log("========== RED（红灯，缺陷未修复） ==========")
		t.Logf("  期望：搜索'性能优化'返回0条结果（doc-2已删除），实际：返回%d条", result2.Total)
		for i, hit := range result2.Hits {
			t.Logf("  结果[%d]: docID=%s, title=%s", i, hit.Document.ID, hit.Document.Title)
		}
		t.Errorf("RED: 搜索'性能优化'应返回0条结果，但返回了%d条", result2.Total)
		return
	}

	t.Log("========== GREEN（绿灯，缺陷已修复） ==========")
	t.Logf("  搜索'倒排索引'返回 %d 条结果，仅包含文档1（doc-1）", result.Total)
	t.Logf("  搜索'性能优化'返回 %d 条结果（doc-2已正确删除）", result2.Total)
}

// indexDoc 对文档进行分词并构建倒排索引。
func indexDoc(t *testing.T, svc *Service, doc *model.Document) {
	t.Helper()
	_, err := svc.BuildIndex(doc)
	if err != nil {
		t.Fatalf("为文档 %s 构建索引失败: %v", doc.ID, err)
	}
}
