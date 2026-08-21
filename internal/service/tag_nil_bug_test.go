package service

import (
	"fmt"
	"testing"

	"benzhi/internal/config"
	"benzhi/internal/store"
)

// TestBugNil009_UploadNewTagAssociation 验证“上传含新标签的文档”时，
// 标签是否被正确创建并关联。
//
// 缺陷存在时：EnsureTag 因“接口持有 nil 指针与 nil 比较为假”而误判标签已存在，
// 导致新标签未被创建，ListTags 中找不到该标签，测试判定为 RED。
// 缺陷修复后：新标签被正确创建并关联，测试判定为 GREEN。
func TestBugNil009_UploadNewTagAssociation(t *testing.T) {
	cfg := config.Get()
	cfg.Storage.DataDir = t.TempDir()
	cfg.Storage.AutoSave = false

	st, err := store.NewStore(cfg.Storage)
	if err != nil {
		t.Fatalf("创建存储失败: %v", err)
	}
	svc := New(st, cfg)

	const newTag = "benzhi-nil-009-new"
	_, err = svc.UploadDocument(&UploadRequest{
		Filename: "doc.txt",
		Data:     []byte("这是一篇用于验证跨文件 nil 接口缺陷的文档正文内容。"),
		Title:    "测试文档",
		Tags:     []string{newTag},
	})
	if err != nil {
		t.Fatalf("上传文档失败: %v", err)
	}

	tags, err := svc.ListTags()
	if err != nil {
		t.Fatalf("列出标签失败: %v", err)
	}

	found := false
	for _, tg := range tags {
		if tg.Name == newTag {
			found = true
			break
		}
	}

	if !found {
		fmt.Println("RED（红灯，缺陷未修复）：上传含新标签的文档后，标签未创建，EnsureTag 误判标签已存在导致关联错误")
		t.Fatalf("期望标签 %q 已被创建并关联，实际 ListTags 中不存在该标签", newTag)
	}

	fmt.Println("GREEN（绿灯，缺陷已修复）：新标签已正确创建并关联")
}
