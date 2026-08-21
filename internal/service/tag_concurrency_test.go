package service_test

import (
	"fmt"
	"sync"
	"testing"

	"benzhi/internal/config"
	"benzhi/internal/service"
	"benzhi/internal/store"
)

// TestBugConcur003_TagCountLostUpdate 验证并发上传多个同标签文档时，
// 标签 Count 是否等于实际关联文档数，从而暴露并发累加丢失更新缺陷。
//
// 缺陷未修复时：
//   - EnsureTag 返回 store 内部共享标签的裸指针（确定性可判）；
//   - 多个 goroutine 在锁外对同一个共享 *model.Tag 做 Count++，后写覆盖先写，
//     最终 Count 小于实际上传文档数。
//   测试打印 RED 并失败。
//
// 缺陷已修复时：store 在写锁内原子累加 Count，EnsureTag 返回副本，
// 最终 Count 等于文档数，测试打印 GREEN 并通过。
func TestBugConcur003_TagCountLostUpdate(t *testing.T) {
	const (
		tagName = "并发标签"
		n       = 100
	)

	cfg := config.Config{
		Storage: config.StorageConfig{
			DataDir:        t.TempDir(),
			DocumentsFile:  "documents.json",
			IndexFile:      "index.json",
			TagsFile:       "tags.json",
			CategoriesFile: "categories.json",
			StatsFile:      "stats.json",
			AutoSave:       false,
		},
		Search: config.SearchConfig{
			MaxPageSize:      100,
			DefaultPageSize:  10,
			BM25K1:           1.2,
			BM25B:            0.75,
		},
		Upload: config.UploadConfig{
			MaxUploadBytes: 10 * 1024 * 1024,
			AllowedFormats: []string{"txt"},
		},
	}

	st, err := store.NewStore(cfg.Storage)
	if err != nil {
		t.Fatalf("初始化存储失败: %v", err)
	}
	svc := service.New(st, cfg)

	// 预创建标签，使后续并发上传都走「已存在标签」的计数自增路径，
	// 从而聚焦于并发累加丢失更新问题本身。
	if _, err := svc.EnsureTag(tagName); err != nil {
		t.Fatalf("预创建标签失败: %v", err)
	}

	// 确定性检查：EnsureTag 是否返回 store 内部共享的裸指针。
	// 缺陷未修复时两次返回同一指针，已修复时返回两个独立副本。
	tagA, errA := svc.EnsureTag(tagName)
	tagB, errB := svc.EnsureTag(tagName)
	if errA != nil || errB != nil {
		t.Fatalf("EnsureTag 失败: %v / %v", errA, errB)
	}
	sharedPointer := tagA == tagB

	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			content := fmt.Sprintf("hello world document number %d", i)
			_, _ = svc.UploadDocument(&service.UploadRequest{
				Filename: fmt.Sprintf("doc-%d.txt", i),
				Data:     []byte(content),
				Title:    fmt.Sprintf("文档-%d", i),
				Tags:     []string{tagName},
			})
		}(i)
	}
	close(start)
	wg.Wait()

	tag, err := st.GetTagByName(tagName)
	if err != nil {
		t.Fatalf("读取标签失败: %v", err)
	}

	if sharedPointer || tag.Count != n {
		fmt.Printf("RED（红灯，缺陷未修复）: 共享裸指针=%v，标签 %q 的 Count=%d，期望=%d\n",
			sharedPointer, tagName, tag.Count, n)
		t.Fatalf("标签关联计数丢失更新: 共享裸指针=%v，期望 Count=%d，实际 Count=%d",
			sharedPointer, n, tag.Count)
	}

	fmt.Printf("GREEN（绿灯，缺陷已修复）: 标签 %q 的 Count=%d，与实际关联文档数一致\n", tagName, tag.Count)
}
