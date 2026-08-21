package service

import (
	"testing"

	"benzhi/internal/config"
	"benzhi/internal/model"
	"benzhi/internal/store"
)

// TestBugOther001_TimestampUnitMixSort 验证“时间戳单位混用导致排序错误”缺陷。
//
// 缺陷未修复时，按上传时间排序（检索与文档列表）会得到升序（最旧在前），
// 与“按上传时间倒序（最新在前）”的契约不符，测试应打印 RED 并失败；
// 缺陷修复后排序恢复倒序，测试应打印 GREEN 并通过。
func TestBugOther001_TimestampUnitMixSort(t *testing.T) {
	cfg := config.Get()
	cfg.Storage.DataDir = t.TempDir()
	cfg.Storage.AutoSave = false

	st, err := store.NewStore(cfg.Storage)
	if err != nil {
		t.Fatalf("创建 Store 失败: %v", err)
	}
	svc := New(st, cfg)

	mk := func(title, content string, uploadTime int64) *model.Document {
		doc, err := svc.CreateDocument(&model.Document{
			Title:      title,
			Content:    content,
			UploadTime: uploadTime,
		})
		if err != nil {
			t.Fatalf("创建文档 %s 失败: %v", title, err)
		}
		if _, err := svc.BuildIndex(doc); err != nil {
			t.Fatalf("为文档 %s 建索引失败: %v", title, err)
		}
		return doc
	}

	// 三篇文档的上传时间故意乱序创建，期望按时间倒序（最新在前）。
	mk("doc-oldest", "apple alpha", 1000)
	mk("doc-newest", "apple beta", 3000)
	mk("doc-middle", "apple gamma", 2000)

	expected := []int64{3000, 2000, 1000}

	// 1) 检索路径：service.sortHits 的 SortByTime 分支。
	res, err := svc.Search(&model.SearchRequest{
		Query:    "apple",
		SortBy:   model.SortByTime,
		Page:     1,
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("检索失败: %v", err)
	}
	searchOrder := make([]int64, 0, len(res.Hits))
	for _, h := range res.Hits {
		searchOrder = append(searchOrder, h.Document.UploadTime)
	}

	// 2) 列表路径：store.ListDocuments。
	docs, err := st.ListDocuments()
	if err != nil {
		t.Fatalf("列出文档失败: %v", err)
	}
	listOrder := make([]int64, 0, len(docs))
	for _, d := range docs {
		listOrder = append(listOrder, d.UploadTime)
	}

	searchOK := equalInt64(searchOrder, expected)
	listOK := equalInt64(listOrder, expected)

	if !searchOK || !listOK {
		t.Errorf("RED（红灯，缺陷未修复）：按上传时间排序错误。检索顺序=%v（期望倒序=%v），列表顺序=%v（期望倒序=%v）",
			searchOrder, expected, listOrder, expected)
		return
	}

	t.Logf("GREEN（绿灯，缺陷已修复）：检索与文档列表均按上传时间倒序返回，顺序=%v", listOrder)
}

// equalInt64 比较两个 int64 切片是否完全一致。
func equalInt64(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
