package service

import (
	"fmt"
	"testing"

	"benzhi/internal/config"
	"benzhi/internal/model"
	"benzhi/internal/store"
)

// TestBugNil008_StatsMissingNilDeref 验证“统计记录缺失时返回 nil 指针被解引用”缺陷。
//
// 触发路径：
//   - internal/store/stats_store.go: GetStats 在统计记录缺失时返回 nil；
//   - internal/service/search_service.go: buildHits 对返回的 nil 指针直接取
//     ViewCount/DownloadCount，触发 nil 指针解引用 panic；
//   - internal/service/stats_service.go: GetDocumentStats 同样对 nil 指针解引用。
//
// 缺陷存在时（RED）：无统计记录的文档参与检索会触发 panic。
// 缺陷修复后（GREEN）：检索正常返回，不触发 panic。
func TestBugNil008_StatsMissingNilDeref(t *testing.T) {
	cfg := config.Get()
	cfg.Storage.DataDir = t.TempDir()
	cfg.Storage.AutoSave = false
	cfg.Search.MaxPageSize = 100
	cfg.Search.DefaultPageSize = 10

	st, err := store.NewStore(cfg.Storage)
	if err != nil {
		t.Fatalf("创建存储失败: %v", err)
	}
	svc := New(st, cfg)

	// 创建一篇文档并建立索引，但故意不为其创建统计记录。
	doc := &model.Document{
		ID:      "doc-nil-008",
		Title:   "无统计记录文档",
		Content: "hello world",
		Format:  "txt",
		Tags:    []string{},
	}
	doc.Normalize()
	if err := st.CreateDocument(doc); err != nil {
		t.Fatalf("创建文档失败: %v", err)
	}
	if _, err := svc.BuildIndex(doc); err != nil {
		t.Fatalf("构建索引失败: %v", err)
	}

	// 通过闭包捕获 panic，避免测试进程直接崩溃，从而打印明确的 RED/GREEN 判定。
	var panicked bool
	var panicValue any
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
				panicValue = r
			}
		}()
		_, _ = svc.Search(&model.SearchRequest{Query: "hello"})
	}()

	if panicked {
		fmt.Println("RED（红灯，缺陷未修复）：无统计记录的文档参与检索触发 nil 指针解引用 panic")
		t.Errorf("RED（红灯，缺陷未修复）：检索无统计记录文档时触发 panic = %v", panicValue)
		return
	}

	fmt.Println("GREEN（绿灯，缺陷已修复）：检索无统计记录文档未触发 panic，正常返回")
}
