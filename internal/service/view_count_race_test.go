package service

import (
	"fmt"
	"sync"
	"testing"

	"benzhi/internal/config"
	"benzhi/internal/model"
	"benzhi/internal/store"
)

// TestBugConcur002_ViewCountLost 验证并发浏览同一文档时浏览次数是否丢失更新。
//
// 缺陷未修复时：非原子读-改-写导致部分并发浏览更新被覆盖，最终计数小于
// 实际请求数，测试打印 RED 并失败。
// 缺陷修复后：并发浏览全部被正确累计，最终计数等于实际请求数，测试打印
// GREEN 并通过。
func TestBugConcur002_ViewCountLost(t *testing.T) {
	st, err := store.NewStore(config.StorageConfig{
		DataDir:  t.TempDir(),
		AutoSave: false,
	})
	if err != nil {
		t.Fatalf("创建 store 失败: %v", err)
	}
	svc := New(st, config.Config{})

	doc, err := svc.CreateDocument(&model.Document{
		Title:   "并发浏览计数测试",
		Content: "用于验证浏览次数在并发读取下是否丢失更新。",
	})
	if err != nil {
		t.Fatalf("创建文档失败: %v", err)
	}

	const workers = 64
	const iterations = 2000
	total := int64(workers * iterations)

	var wg sync.WaitGroup
	start := make(chan struct{})

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < iterations; j++ {
				_, _ = svc.GetDocument(doc.ID)
			}
		}()
	}

	close(start)
	wg.Wait()

	stats, err := svc.GetDocumentStats(doc.ID)
	if err != nil {
		t.Fatalf("读取统计失败: %v", err)
	}
	final := stats.ViewCount
	expected := total

	if final == expected {
		fmt.Printf("GREEN（绿灯，缺陷已修复）：并发浏览 %d 次，最终计数 %d，无丢失\n", total, final)
		return
	}

	fmt.Printf("RED（红灯，缺陷未修复）：并发浏览 %d 次，最终计数 %d，丢失 %d 次更新\n", total, final, expected-final)
	t.Fatalf("浏览计数丢失：期望 %d，实际 %d", expected, final)
}
