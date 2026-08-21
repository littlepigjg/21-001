package service

import (
	"testing"

	"benzhi/internal/config"
	"benzhi/internal/model"
	"benzhi/internal/store"
)

// TestBugError020_DeleteIgnoredCleanupError 验证“删除时忽略清理错误导致索引残留”缺陷。
//
// 判定逻辑：
//   - 删除文档后，如果检索仍能命中该文档（索引残留），说明缺陷未修复，输出 RED 并失败；
//   - 删除文档后，如果检索不再命中该文档（索引已正确清理），说明缺陷已修复，输出 GREEN。
func TestBugError020_DeleteIgnoredCleanupError(t *testing.T) {
	cfg := config.Get()
	cfg.Storage.DataDir = t.TempDir()
	cfg.Storage.AutoSave = false

	st, err := store.NewStore(cfg.Storage)
	if err != nil {
		t.Fatalf("创建存储失败: %v", err)
	}
	svc := New(st, cfg)

	// 1. 创建一篇文档。
	doc, err := svc.CreateDocument(&model.Document{
		Title:   "删除残留复现文档",
		Content: "zebraplanet alpha beta gamma",
	})
	if err != nil {
		t.Fatalf("创建文档失败: %v", err)
	}

	// 2. 为文档建立倒排索引。
	if _, err := svc.BuildIndex(doc); err != nil {
		t.Fatalf("构建索引失败: %v", err)
	}

	// 3. 删除文档（此处应成功返回，但缺陷会导致内部清理失败被吞掉）。
	if err := svc.DeleteDocument(doc.ID); err != nil {
		t.Fatalf("删除文档失败: %v", err)
	}

	// 4. 检索唯一词，验证是否仍能命中已删除的文档。
	res, err := svc.Search(&model.SearchRequest{Query: "zebraplanet"})
	if err != nil {
		t.Fatalf("检索失败: %v", err)
	}

	if res.Total > 0 {
		t.Errorf("RED（红灯，缺陷未修复）：删除文档后检索仍命中 %d 篇已删文档，索引残留未被清理", res.Total)
		return
	}
	t.Logf("GREEN（绿灯，缺陷已修复）：删除文档后检索命中 0 篇，索引已正确清理")
}
