package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"benzhi/internal/config"
	"benzhi/internal/model"
	"benzhi/internal/store"
)

func TestBugDefer026_ImportHandleExhaustion(t *testing.T) {
	dataDir, err := os.MkdirTemp("", "benzhi-defer-026-")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(dataDir)

	st, err := store.NewStore(config.StorageConfig{
		DataDir:        dataDir,
		DocumentsFile:  "documents.json",
		IndexFile:      "index.json",
		TagsFile:       "tags.json",
		CategoriesFile: "categories.json",
		StatsFile:      "stats.json",
		AutoSave:       false,
	})
	if err != nil {
		t.Fatalf("创建 Store 失败: %v", err)
	}

	svc := New(st, config.Config{})

	const n = 50
	docs := make([]*model.Document, 0, n)
	for i := 0; i < n; i++ {
		docs = append(docs, &model.Document{
			Title:   fmt.Sprintf("批量导入文档 %d", i),
			Content: fmt.Sprintf("这是第 %d 篇用于验证文件句柄耗尽的文档正文内容。", i),
		})
	}

	payload, err := json.Marshal(docs)
	if err != nil {
		t.Fatalf("序列化导入数据失败: %v", err)
	}

	imported, err := svc.ImportDocumentsJSON(bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("导入失败: %v", err)
	}
	if imported != n {
		t.Fatalf("导入数量不符: 期望 %d, 实际 %d", n, imported)
	}

	peak := st.PeakOpenContentHandles()
	if peak <= 1 {
		fmt.Printf("GREEN（绿灯，缺陷已修复）：批量导入 %d 篇文档期间，同时打开的原文文件句柄峰值为 %d（<= 1）\n", n, peak)
		return
	}

	fmt.Printf("RED（红灯，缺陷未修复）：批量导入 %d 篇文档期间，同时打开的原文文件句柄峰值为 %d（预期 <= 1），说明句柄在循环内被 defer 堆积直到函数结束才释放\n", n, peak)
	t.Fatalf("文件句柄在循环内被 defer 堆积，峰值=%d", peak)
}
