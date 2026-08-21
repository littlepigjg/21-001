package service

import (
	"fmt"
	"testing"

	"benzhi/internal/config"
	"benzhi/internal/model"
	"benzhi/internal/store"
)

// bugSlice014HugePage 是 int 的最大值，用于模拟用户请求“超大页码”。
const bugSlice014HugePage = int(^uint(0) >> 1)

// newBugSlice014Service 构造一个带少量文档的 Service，供分页缺陷验证使用。
func newBugSlice014Service(t *testing.T) *Service {
	t.Helper()

	st, err := store.NewStore(config.StorageConfig{
		DataDir:        t.TempDir(),
		DocumentsFile:  "documents.json",
		IndexFile:      "index.json",
		TagsFile:       "tags.json",
		CategoriesFile: "categories.json",
		StatsFile:      "stats.json",
		AutoSave:       false,
	})
	if err != nil {
		t.Fatalf("创建存储失败: %v", err)
	}

	svc := New(st, config.Config{
		Search: config.SearchConfig{
			MaxPageSize:     100,
			DefaultPageSize: 10,
		},
	})

	for i := 0; i < 5; i++ {
		doc := &model.Document{
			ID:      fmt.Sprintf("doc-%d", i),
			Title:   fmt.Sprintf("标题-%d", i),
			Content: fmt.Sprintf("正文内容 %d", i),
			Tags:    []string{},
			Format:  model.FormatTXT,
		}
		if err := st.CreateDocument(doc); err != nil {
			t.Fatalf("创建文档失败: %v", err)
		}
	}

	return svc
}

// runWithRecover 执行 fn，返回是否触发 panic。
func runWithRecover(t *testing.T, fn func()) (panicked bool) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			panicked = true
			t.Logf("捕获 panic: %v", r)
		}
	}()
	fn()
	return false
}

// TestBugSlice014_PageOffsetNotValidated 验证“分页偏移未校验导致切片越界”缺陷。
//
// 缺陷存在时：超大页码下 (page-1)*pageSize 溢出为负数，切片时触发
// "slice bounds out of range" panic，测试打印 RED 并失败。
// 缺陷修复后：分页窗口被正确钳制，两个路径均正常返回，测试打印 GREEN 并通过。
func TestBugSlice014_PageOffsetNotValidated(t *testing.T) {
	svc := newBugSlice014Service(t)

	hits := []model.SearchHit{
		{Document: model.Document{ID: "a", Title: "a"}},
		{Document: model.Document{ID: "b", Title: "b"}},
	}

	searchPanicked := runWithRecover(t, func() {
		_ = svc.paginate(hits, bugSlice014HugePage, 10)
	})

	listPanicked := runWithRecover(t, func() {
		_, _, _ = svc.ListDocuments(bugSlice014HugePage, 10)
	})

	if searchPanicked || listPanicked {
		fmt.Println("RED（红灯，缺陷未修复）：超大页码下分页切片越界触发 slice bounds out of range panic")
		t.Fatalf("RED（红灯，缺陷未修复）：paginate panic=%v, ListDocuments panic=%v", searchPanicked, listPanicked)
	}

	fmt.Println("GREEN（绿灯，缺陷已修复）：超大页码下分页均正常返回，未触发越界 panic")
}
