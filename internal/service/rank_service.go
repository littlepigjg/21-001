package service

import (
	"math"
	"unicode/utf8"

	"benzhi/internal/model"
)

// RankService 相关方法负责实现文档相关度评分（BM25）。
//
// BM25 是经典的概率检索排序算法，综合考虑词频、文档长度与逆文档频率。
// 相关度分数仅用于 relevance 排序，不影响其他排序方式。

// analyzeQuery 对查询语句分词并去重，返回查询词项列表。
func (s *Service) analyzeQuery(query string) []string {
	if query == "" {
		return nil
	}
	seen := make(map[string]struct{})
	var out []string
	for _, tok := range s.tokenizer.Tokenize(query) {
		if _, ok := seen[tok]; ok {
			continue
		}
		seen[tok] = struct{}{}
		out = append(out, tok)
	}
	return out
}

// idf 计算逆文档频率分量。
//
// N 为语料文档总数，n 为包含该词项的文档数。
func (s *Service) idf(n, N int) float64 {
	if N <= 0 {
		return 0
	}
	return math.Log(1 + (float64(N-n)+0.5)/(float64(n)+0.5))
}

// docLen 返回文档长度（以字符数近似）。
func docLen(doc *model.Document) float64 {
	if doc == nil {
		return 0
	}
	return float64(utf8.RuneCountInString(doc.Content))
}

// averageDocLen 计算文档集合的平均长度。
func averageDocLen(docs []*model.Document) float64 {
	if len(docs) == 0 {
		return 1
	}
	var sum float64
	for _, d := range docs {
		sum += docLen(d)
	}
	return sum / float64(len(docs))
}

// scoreDocument 计算单个文档相对查询的 BM25 分数。
//
// termFreq 为查询词在该文档中的词频，docFreq 为包含该词项的文档总数，
// N 为语料文档总数，avgdl 为平均文档长度。
func (s *Service) scoreDocument(queryTerms []string, doc *model.Document, docFreq map[string]int, N int, avgdl float64) float64 {
	tf := s.tokenizer.CountTerms(doc.Content)
	dl := docLen(doc)

	k1 := s.cfg.Search.BM25K1
	b := s.cfg.Search.BM25B

	var score float64
	for _, term := range queryTerms {
		f := float64(tf[term])
		if f <= 0 {
			continue
		}
		idf := s.idf(docFreq[term], N)
		numerator := f * (k1 + 1)
		denominator := f + k1*(1-b+b*dl/avgdl)
		if denominator == 0 {
			continue
		}
		score += idf * numerator / denominator
	}
	return score
}
