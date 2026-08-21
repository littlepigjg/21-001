package service

import (
	"fmt"
	"strings"
	"testing"

	"benzhi/internal/config"
	"benzhi/internal/store"
)

// TestBugNil010_ImportDocumentsJSONNilElement 验证导入含 nil 元素的 JSON 数组时，
// 服务层与存储层是否会对 nil 元素做校验，避免直接解引用触发 panic。
//
// 缺陷未修复时：ImportDocumentsJSON 不校验 d == nil，直接把 nil 元素传给
// store.CreateDocument；CreateDocument 也不校验 doc == nil，直接解引用 doc.Title，
// 触发 panic nil pointer dereference -> 判定 RED，测试失败。
//
// 缺陷修复后：nil 元素在服务层/存储层被跳过或拒绝，不再 panic -> 判定 GREEN，测试通过。
func TestBugNil010_ImportDocumentsJSONNilElement(t *testing.T) {
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
	svc := New(st, config.Get())

	var panicValue interface{}
	var importErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicValue = r
			}
		}()
		_, importErr = svc.ImportDocumentsJSON(strings.NewReader(`[null]`))
	}()

	if panicValue != nil {
		fmt.Println("RED（红灯，缺陷未修复）：导入含 nil 元素的 JSON 数组触发 panic:", panicValue)
		t.Fatalf("缺陷存在：ImportDocumentsJSON 未校验 nil 元素，直接解引用导致 panic: %v", panicValue)
	}
	if importErr != nil {
		fmt.Println("RED（红灯，缺陷未修复）：导入含 nil 元素的 JSON 数组返回错误:", importErr)
		t.Fatalf("缺陷存在：导入含 nil 元素返回错误: %v", importErr)
	}

	fmt.Println("GREEN（绿灯，缺陷已修复）：导入含 nil 元素的 JSON 数组被安全跳过，未发生 panic")
}
