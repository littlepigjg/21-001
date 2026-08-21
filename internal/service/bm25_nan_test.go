package service

import (
	"fmt"
	"math"
	"testing"

	"benzhi/internal/config"
	"benzhi/internal/model"
	"benzhi/internal/store"
)

// TestBugOther030_BM25NaNSort 验证 BM25 分母除零导致 NaN 评分的缺陷。
//
// 当候选文档正文为空时，平均文档长度 avgdl 退化为 0，scoreDocument 在计算
// 长度归一化 dl/avgdl 时出现 0/0，最终把 NaN 写入 SearchHit.Score，导致
// relevance 排序错乱。
//
// 缺陷未修复时本测试会打印 RED（红灯）并使 go test 失败；
// 缺陷修复后本测试打印 GREEN（绿灯）并通过。
func TestBugOther030_BM25NaNSort(t *testing.T) {
	// 使用临时目录构造一个真实的内存存储，避免落盘与外部依赖。
	st, err := store.NewStore(config.StorageConfig{
		DataDir:  t.TempDir(),
		AutoSave: false,
	})
	if err != nil {
		t.Fatalf("创建存储失败: %v", err)
	}

	// 注册一篇正文为空的文档。空正文无法产生任何倒排词项，
	// 因此命中评分的词频 f 为 0，同时 docLen 为 0。
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

	// 直接走 buildHits -> scoreDocument 的完整评分链路。
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
