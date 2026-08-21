package store

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"benzhi/internal/config"
)

// TestBugNil007_NilMapWriteAfterEmptyIndexLoad 复现跨文件 nil map 缺陷：
// 磁盘上已存在一个 terms 为 null 的“空索引”，Load 加载后未初始化 map，
// AddPosting 直接向 nil map 写入词项，应触发 "assignment to entry in nil map" panic。
//
// 判定规则：
//   - 缺陷未修复（AddPosting panic） -> 测试失败，输出 RED（红灯）。
//   - 缺陷已修复（AddPosting 正常写入）-> 测试通过，输出 GREEN（绿灯）。
func TestBugNil007_NilMapWriteAfterEmptyIndexLoad(t *testing.T) {
	dataDir := t.TempDir()

	// 模拟磁盘上已持久化的空索引：terms 字段为 null（未初始化）。
	idxPath := filepath.Join(dataDir, "index.json")
	if err := os.WriteFile(idxPath, []byte(`{"version":1,"terms":null,"doc_count":0}`), 0o644); err != nil {
		t.Fatalf("写入空索引文件失败: %v", err)
	}

	cfg := config.StorageConfig{
		DataDir:        dataDir,
		DocumentsFile:  "documents.json",
		IndexFile:      "index.json",
		TagsFile:       "tags.json",
		CategoriesFile: "categories.json",
		StatsFile:      "stats.json",
		AutoSave:       false,
	}

	st, err := NewStore(cfg)
	if err != nil {
		t.Fatalf("NewStore 失败: %v", err)
	}
	if err := st.Load(); err != nil {
		t.Fatalf("Load 失败: %v", err)
	}

	// 触发跨文件缺陷：加载空索引后向 nil map 写入词项。
	var panicMsg string
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicMsg = fmt.Sprint(r)
			}
		}()
		st.AddPosting("hello", "doc-1", []int{0})
	}()

	if panicMsg != "" {
		fmt.Println("RED（红灯，缺陷未修复）：加载空索引后 AddPosting 触发 panic ->", panicMsg)
		t.Fatalf("缺陷仍存在：加载空索引后写入词项触发 panic: %v", panicMsg)
	}

	pl := st.GetPostingList("hello")
	if pl.DocFreq != 1 || len(pl.Postings) != 1 || pl.Postings[0].DocID != "doc-1" {
		fmt.Printf("RED（红灯，缺陷未修复）：词项写入结果不符合预期: %+v\n", pl)
		t.Fatalf("词项写入结果不符合预期: %+v", pl)
	}

	fmt.Println("GREEN（绿灯，缺陷已修复）：加载空索引后写入词项正常，未触发 panic")
}
