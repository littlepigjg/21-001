package service

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"

	"benzhi/internal/config"
	"benzhi/internal/model"
	"benzhi/internal/store"
)

// TestBugConcur004_HotRankingRace 验证「排序读取统计与自增并发导致热度值不一致」缺陷。
//
// 由于 Go 运行时的 "concurrent map read and map write" 属于不可 recover 的致命错误，
// 这里采用父进程-子进程方式：父进程重新执行本测试二进制中的子用例，若子进程崩溃并
// 输出 "concurrent map"，则判定为 RED（缺陷未修复）；否则为 GREEN（缺陷已修复）。
func TestBugConcur004_HotRankingRace(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run", "^TestBugConcur004_Child$")
	out, _ := cmd.CombinedOutput()

	if strings.Contains(string(out), "concurrent map") {
		fmt.Println("RED（红灯，缺陷未修复）：热度排序读取统计与浏览自增存在并发 map 读写数据竞争")
		t.Fatalf("检测到并发 map 读写缺陷，子进程关键输出：\n%s", string(out))
	}

	fmt.Println("GREEN（绿灯，缺陷已修复）：并发检索（按热度排序）与高频浏览未发生数据竞争")
}

// TestBugConcur004_Child 是子进程用例，运行会触发数据竞争的并发负载。
func TestBugConcur004_Child(t *testing.T) {
	runHotRankingRacyWorkload()
}

// runHotRankingRacyWorkload 同时并发执行「按热度排序的检索（读 stats map）」与
// 「高频浏览文档（写 stats map）」，触发 GetStats 无锁读取与 IncrementView/EnsureStats
// 写操作之间的数据竞争。
func runHotRankingRacyWorkload() {
	dataDir, err := os.MkdirTemp("", "benzhi-hot-race-*")
	if err != nil {
		fmt.Println("创建临时目录失败:", err)
		return
	}
	defer os.RemoveAll(dataDir)

	cfg := config.Config{
		Storage: config.StorageConfig{
			DataDir:        dataDir,
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
			AllowedFormats: []string{"txt"},
		},
	}

	st, err := store.NewStore(cfg.Storage)
	if err != nil {
		fmt.Println("构建 Store 失败:", err)
		return
	}
	svc := New(st, cfg)

	// 预置若干可被检索到的文档，保证 Search 会走到 buildHits 读取 stats。
	for i := 0; i < 4; i++ {
		doc := &model.Document{
			Title:   "t",
			Content: fmt.Sprintf("concurrent map race test content %d", i),
		}
		created, err := svc.CreateDocument(doc)
		if err != nil {
			fmt.Println("创建文档失败:", err)
			return
		}
		if _, err := svc.BuildIndex(created); err != nil {
			fmt.Println("构建索引失败:", err)
			return
		}
	}

	const writers = 16
	const readers = 16
	const iters = 200

	var wg sync.WaitGroup

	// 写路径：高频浏览文档。IncrementView 会 EnsureStats 并为新 docID 写入 stats map。
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < iters; j++ {
				svc.IncrementView(fmt.Sprintf("doc-hot-%d-%d", n, j))
			}
		}(i)
	}

	// 读路径：并发按热度排序检索，间接调用 buildHits -> GetStats 无锁读取 stats map。
	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iters; j++ {
				_, _ = svc.Search(&model.SearchRequest{Query: "concurrent", SortBy: "hot"})
			}
		}()
	}

	wg.Wait()
}
