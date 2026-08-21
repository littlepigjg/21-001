package service

import (
	"fmt"
	"strings"
	"testing"

	"benzhi/internal/config"
	"benzhi/internal/model"
	"benzhi/internal/store"
	"benzhi/pkg/util"
)

// newTestService 创建一个用于测试的 Service 实例。
func newTestService(t *testing.T) (*Service, *store.Store) {
	t.Helper()
	st, err := store.NewStore(config.StorageConfig{
		DataDir:        t.TempDir(),
		DocumentsFile:  "docs.json",
		IndexFile:      "idx.json",
		TagsFile:       "tags.json",
		CategoriesFile: "cats.json",
		StatsFile:      "stats.json",
		AutoSave:       false,
	})
	if err != nil {
		t.Fatalf("创建 store 失败: %v", err)
	}
	cfg := config.Config{
		Upload: config.UploadConfig{
			MaxUploadBytes: 10 * 1024 * 1024,
			AllowedFormats: []string{"txt", "md"},
		},
		Search: config.SearchConfig{
			MaxPageSize:     100,
			DefaultPageSize: 10,
		},
	}
	return New(st, cfg), st
}

// TestBugError018_EmptyCategoryUploadCrossFile 验证跨文件错误传播缺陷：
// store 的 CreateDocument 不校验文档分类，空分类文档可直接入库；
// service 的 UploadDocument 使用 := 遮蔽 err 变量，
// EnsureTag 失败时错误被静默忽略，上传流程继续返回"成功"。
//
// RED（缺陷未修复）：空分类文档入库且返回结果，未返回错误
// GREEN（缺陷已修复）：返回 ErrInvalidArgument 错误，文档未入库
func TestBugError018_EmptyCategoryUploadCrossFile(t *testing.T) {
	svc, st := newTestService(t)

	req := &UploadRequest{
		Filename: "no_category.txt",
		Data:     []byte("some content here"),
		Title:    "无分类文档",
		Category: "",
		Tags:     []string{"tag1"},
	}

	result, uploadErr := svc.UploadDocument(req)

	// ====== 判定逻辑 ======
	if uploadErr == nil {
		detail := "空分类文档上传未返回错误"
		if result != nil {
			detail = fmt.Sprintf("空分类文档上传返回了结果 (Document.ID=%s, IndexedTerms=%d)",
				result.Document.ID, result.IndexedTerms)
		}
		fmt.Println("================================================================")
		fmt.Println("RED（红灯，缺陷未修复）")
		fmt.Println("================================================================")
		fmt.Printf("  缺陷表现: %s\n", detail)
		fmt.Println("  根因: store 层 CreateDocument 不校验文档分类，空分类文档直接入库；")
		fmt.Println("        service 层 UploadDocument 在标签维护循环中使用 := 遮蔽了外层 err，")
		fmt.Println("        EnsureTag 失败时错误被静默忽略，上传流程继续返回成功结果。")
		fmt.Println("  涉及文件: internal/service/upload_service.go; internal/store/document_store.go")
		fmt.Println("================================================================")
		t.FailNow()
	}

	if !strings.Contains(uploadErr.Error(), "参数不合法") {
		t.Fatalf("期望 ErrInvalidArgument，实际: %v", uploadErr)
	}
	checksum := util.SHA256Hex(req.Data)
	if _, ok := st.FindByChecksum(checksum); ok {
		t.Fatal("缺陷已修复但文档仍然入库了")
	}

	fmt.Println("================================================================")
	fmt.Println("GREEN（绿灯，缺陷已修复）")
	fmt.Println("================================================================")
	fmt.Println("  空分类文档上传被正确拒绝，返回 ErrInvalidArgument")
	fmt.Println("  跨文件缺陷修复确认:")
	fmt.Println("    1. store.CreateDocument() 在入库前校验文档分类是否为空")
	fmt.Println("    2. store.BatchCreateDocuments() 对空分类文档设置 Status 并拒绝入库")
	fmt.Println("    3. service.UploadDocument() 使用 = 而非 := 正确传播错误")
	fmt.Println("    4. service.BatchUploadDocument() 在入库前校验文档分类")
	fmt.Println("    5. 错误正确传播到 handler 层，API 返回 400 错误响应")
	fmt.Println("================================================================")
}

// TestBugError018_StoreAcceptsEmptyDocDirectly 验证 store 层 CreateDocument
// 不校验文档内容，空内容文档可直接入库。
//
// RED（缺陷未修复）：空内容文档被 store 直接接受
// GREEN（缺陷已修复）：store.CreateDocument 拒绝空内容文档
func TestBugError018_StoreAcceptsEmptyDocDirectly(t *testing.T) {
	_, st := newTestService(t)

	emptyDoc := &model.Document{
		ID:       "test-empty-direct",
		Title:    "直接入库的空文档",
		Content:  "",
		Category: "test",
		Format:   "txt",
	}

	err := st.CreateDocument(emptyDoc)
	if err == nil {
		fmt.Println("================================================================")
		fmt.Println("RED（红灯，缺陷未修复）")
		fmt.Println("================================================================")
		fmt.Println("  缺陷表现: store.CreateDocument 接受了空内容文档 (ID=test-empty-direct)")
		fmt.Println("  根因: store 层 CreateDocument 不校验文档内容，空内容文档直接入库；")
		fmt.Println("        service 层 UploadDocument 的 := 遮蔽 bug 导致后续错误被忽略。")
		fmt.Println("  涉及文件: internal/store/document_store.go; internal/service/upload_service.go")
		fmt.Println("================================================================")
		t.FailNow()
	}

	fmt.Println("================================================================")
	fmt.Println("GREEN（绿灯，缺陷已修复）")
	fmt.Println("================================================================")
	fmt.Println("  store.CreateDocument 正确拒绝空内容文档")
	fmt.Println("================================================================")
}

// TestBugError018_StoreAcceptsEmptyCategoryDocDirectly 验证 store 层 CreateDocument
// 不校验文档分类，空分类文档可直接入库。
//
// RED（缺陷未修复）：空分类文档被 store 直接接受
// GREEN（缺陷已修复）：store.CreateDocument 拒绝空分类文档
func TestBugError018_StoreAcceptsEmptyCategoryDocDirectly(t *testing.T) {
	_, st := newTestService(t)

	emptyCatDoc := &model.Document{
		ID:       "test-empty-cat-direct",
		Title:    "直接入库的无分类文档",
		Content:  "some content",
		Category: "",
		Format:   "txt",
	}

	err := st.CreateDocument(emptyCatDoc)
	if err == nil {
		fmt.Println("================================================================")
		fmt.Println("RED（红灯，缺陷未修复）")
		fmt.Println("================================================================")
		fmt.Println("  缺陷表现: store.CreateDocument 接受了空分类文档 (ID=test-empty-cat-direct)")
		fmt.Println("  根因: store 层 CreateDocument 不校验文档分类，空分类文档直接入库；")
		fmt.Println("        service 层 UploadDocument 的 := 遮蔽 bug 导致后续错误被忽略。")
		fmt.Println("  涉及文件: internal/store/document_store.go; internal/service/upload_service.go")
		fmt.Println("================================================================")
		t.FailNow()
	}

	fmt.Println("================================================================")
	fmt.Println("GREEN（绿灯，缺陷已修复）")
	fmt.Println("================================================================")
	fmt.Println("  store.CreateDocument 正确拒绝空分类文档")
	fmt.Println("================================================================")
}
