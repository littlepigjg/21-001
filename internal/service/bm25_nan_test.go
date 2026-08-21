package service

import (
	"fmt"
	"math"
	"testing"

	"benzhi/internal/config"
	"benzhi/internal/model"
	"benzhi/internal/store"
)

func TestBugOther030_BM25NaNSort(t *testing.T) {
	st, err := store.NewStore(config.StorageConfig{
		DataDir:  t.TempDir(),
		AutoSave: false,
	})
	if err != nil {
		t.Fatalf("创建存储失败: %v", err)
	}

	emptyDoc := &model.Document{ID: "empty-doc", Title: "empty", Content: ""}
	if err := st.CreateDocument(emptyDoc); err != nil {
		t.Fatalf("写入空文档失败: %v", err)
	}

	cfg := config.Config{
		Search: config.SearchConfig{
			BM25K1: 1.2,
			BM25B:  0.75,
		},
	}
	svc := New(st, cfg)

	docs := []*model.Document{{ID: "empty-doc", Title: "empty", Content: ""}}
	hits := svc.buildHits(docs, []string{"hello"}, model.SortByRelevance)

	nonFinite := false
	for _, h := range hits {
		if math.IsNaN(h.Score) || math.IsInf(h.Score, 0) {
			nonFinite = true
		}
		fmt.Printf("doc=%s score=%v\n", h.Document.ID, h.Score)
	}

	if nonFinite {
		fmt.Println("判定结果：RED（红灯，缺陷未修复）—— BM25 评分出现 NaN/Inf，排序错乱")
		t.Errorf("检测到非有限评分（NaN/Inf），BM25 分母除零缺陷仍存在")
		return
	}

	fmt.Println("判定结果：GREEN（绿灯，缺陷已修复）—— BM25 评分均为有限值，无 NaN")
}
