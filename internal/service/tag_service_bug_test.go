package service

import (
	"fmt"
	"testing"

	"benzhi/internal/config"
	"benzhi/internal/model"
	"benzhi/internal/store"
)

// TestBugSlice015_RemoveTagDeleteResidue 验证删除标签后文档是否残留已删除的标签。
//
// 缺陷修复前：DeleteTag 复用共享底层数组过滤标签，RemoveTag 又按原始长度
// 截断过滤结果，导致被删除标签残留在文档的 Tags 尾部，测试判定为 RED。
// 缺陷修复后：删除标签后文档不再包含该标签，测试判定为 GREEN。
func TestBugSlice015_RemoveTagDeleteResidue(t *testing.T) {
	st, err := store.NewStore(config.StorageConfig{DataDir: "", AutoSave: false})
	if err != nil {
		t.Fatalf("创建存储失败: %v", err)
	}
	svc := New(st, config.Config{})

	tag, err := svc.CreateTag("shared-tag")
	if err != nil {
		t.Fatalf("创建标签失败: %v", err)
	}

	doc1, err := svc.CreateDocument(&model.Document{
		Title:   "文档一",
		Content: "这是文档一的正文内容",
		Tags:    []string{"keep-a", "shared-tag"},
	})
	if err != nil {
		t.Fatalf("创建文档一失败: %v", err)
	}

	doc2, err := svc.CreateDocument(&model.Document{
		Title:   "文档二",
		Content: "这是文档二的正文内容",
		Tags:    []string{"keep-b", "shared-tag"},
	})
	if err != nil {
		t.Fatalf("创建文档二失败: %v", err)
	}

	if err := svc.DeleteTag(tag.ID); err != nil {
		t.Fatalf("删除标签失败: %v", err)
	}

	got1, err := svc.GetDocument(doc1.ID)
	if err != nil {
		t.Fatalf("读取文档一失败: %v", err)
	}
	got2, err := svc.GetDocument(doc2.ID)
	if err != nil {
		t.Fatalf("读取文档二失败: %v", err)
	}

	residue1 := got1.HasTag("shared-tag")
	residue2 := got2.HasTag("shared-tag")
	empty1 := hasEmptyTag(got1.Tags)
	empty2 := hasEmptyTag(got2.Tags)
	exact1 := equalStringSlices(got1.Tags, []string{"keep-a"})
	exact2 := equalStringSlices(got2.Tags, []string{"keep-b"})

	if residue1 || residue2 || empty1 || empty2 || !exact1 || !exact2 {
		fmt.Println("RED（红灯，缺陷未修复）：删除标签后文档仍残留已删除标签")
		t.Fatalf(
			"RED（红灯，缺陷未修复）：删除标签后文档仍残留已删除标签；"+
				"doc1.Tags=%v (residue=%v, empty=%v, exact=%v), doc2.Tags=%v (residue=%v, empty=%v, exact=%v)",
			got1.Tags, residue1, empty1, exact1, got2.Tags, residue2, empty2, exact2,
		)
	}

	fmt.Println("GREEN（绿灯，缺陷已修复）：删除标签后文档不再残留已删除标签")
}

// hasEmptyTag 判断标签列表中是否包含空字符串。
func hasEmptyTag(tags []string) bool {
	for _, tg := range tags {
		if tg == "" {
			return true
		}
	}
	return false
}

// equalStringSlices 判断两个字符串切片是否完全一致。
func equalStringSlices(a, b []string) bool {
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
