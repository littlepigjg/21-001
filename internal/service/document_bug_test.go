package service

import (
	"fmt"
	"reflect"
	"testing"

	"benzhi/internal/config"
	"benzhi/internal/model"
	"benzhi/internal/store"
)

// newBugTestService 构建一个不落盘、用于缺陷验证的 Service 实例。
func newBugTestService(t *testing.T) *Service {
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
		t.Fatalf("创建存储实例失败: %v", err)
	}

	cfg := config.Get()
	cfg.Storage.AutoSave = false
	return New(st, cfg)
}

// TestBugSlice013_TagPollution 复现“返回文档复用内部 Tags 切片导致污染”的缺陷。
//
// 缺陷存在时（RED）：更新文档标签会就地修改共享的底层数组，导致更新前获取的
// 文档引用其标签被意外篡改。
// 缺陷修复后（GREEN）：更新前获取的文档标签保持独立，不受后续更新影响。
func TestBugSlice013_TagPollution(t *testing.T) {
	svc := newBugTestService(t)

	created, err := svc.CreateDocument(&model.Document{
		Title:    "原始文档",
		Content:  "用于复现标签污染缺陷的正文内容。",
		Category: "tech",
		Tags:     []string{"alpha", "beta"},
	})
	if err != nil {
		t.Fatalf("创建文档失败: %v", err)
	}

	// 更新标签前先取一次文档，该引用应当保持独立。
	before, err := svc.GetDocument(created.ID)
	if err != nil {
		t.Fatalf("获取文档失败: %v", err)
	}

	// 更新文档标签。
	if _, err := svc.UpdateDocument(created.ID, &model.Document{
		Tags: []string{"gamma"},
	}); err != nil {
		t.Fatalf("更新文档失败: %v", err)
	}

	// 再次获取文档，确认标签按预期被更新。
	after, err := svc.GetDocument(created.ID)
	if err != nil {
		t.Fatalf("再次获取文档失败: %v", err)
	}

	wantBefore := []string{"alpha", "beta"}
	wantAfter := []string{"gamma"}

	if !reflect.DeepEqual(after.Tags, wantAfter) {
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Fatalf("RED（红灯，缺陷未修复）：更新后标签不符合预期 got=%v want=%v", after.Tags, wantAfter)
	}

	if !reflect.DeepEqual(before.Tags, wantBefore) {
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Fatalf("RED（红灯，缺陷未修复）：更新前获取的文档标签被污染 got=%v want=%v", before.Tags, wantBefore)
	}

	fmt.Println("GREEN（绿灯，缺陷已修复）")
}
