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

// TestBugConcur001_InvertedIndexRace 验证倒排索引 map 在并发建索引与检索时的数据竞争。
//
// 由于 Go 运行时的 "concurrent map read and map write" 属于不可 recover 的致命错误，
// 这里采用父进程-子进程方式：父进程重新执行本测试二进制中的子用例，若子进程崩溃
// 并输出 "concurrent map"，则判定为 RED（缺陷未修复）；否则为 GREEN（缺陷已修复）。
func TestBugConcur001_InvertedIndexRace(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run", "^TestBugConcur001_Child$")
	out, _ := cmd.CombinedOutput()

	if strings.Contains(string(out), "concurrent map") {
		fmt.Println("RED（红灯，缺陷未修复）：倒排索引 map 存在并发读写数据竞争")
		t.Fatalf("检测到并发 map 读写缺陷，子进程关键输出：\n%s", string(out))
	}

	fmt.Println("GREEN（绿灯，缺陷已修复）：并发建索引与检索未发生数据竞争")
}

// TestBugConcur001_Child 是子进程用例，运行会触发数据竞争的并发负载。
func TestBugConcur001_Child(t *testing.T) {
	runRacyWorkload()
}

// runRacyWorkload 同时并发执行建索引（写 map）与检索（读 map），触发数据竞争。
func runRacyWorkload() {
	dataDir, err := os.MkdirTemp("", "benzhi-race-*")
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

	const writers = 16
	const readers = 16
	const iters = 100

	var wg sync.WaitGroup

	// 写路径：多个上传 goroutine 并发建索引，间接调用 store.MergePostingList 无锁写倒排索引 map。
	// 通过 writeMu 串行化写索引操作，与读路径形成稳定的读写竞争，复现 "concurrent map read and map write"。
	var writeMu sync.Mutex
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < iters; j++ {
				doc := &model.Document{
					ID:      fmt.Sprintf("doc-%d-%d", n, j),
					Title:   "t",
					Content: fmt.Sprintf("concurrent map race test content %d %d", n, j),
				}
				writeMu.Lock()
				_, _ = svc.BuildIndex(doc)
				writeMu.Unlock()
			}
		}(i)
	}

	// 读路径：并发检索，间接调用 collectCandidates 通过 store.Terms() 无锁读 map。
	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iters; j++ {
				_, _ = svc.Search(&model.SearchRequest{Query: "concurrent"})
			}
		}()
	}

	wg.Wait()
}
