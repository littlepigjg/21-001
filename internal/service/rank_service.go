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

// docLengthStats 保存候选文档集合的长度统计信息，供 BM25 长度归一化使用。
type docLengthStats struct {
	// count 是候选文档数量。
	count int
	// sum 是所有候选文档正文长度之和。
	sum float64
	// avgdl 是候选文档平均长度。
	avgdl float64
}

// computeDocLengthStats 统计候选文档集合的长度信息。
//
// 该函数直接使用 sum/count 计算平均长度：当候选集合为空，或集合内所有
// 文档正文都为空时，avgdl 会退化为 0。后续 BM25 长度归一化在 avgdl 为 0
// 的情况下会出现除零，导致评分产生 NaN。
func computeDocLengthStats(docs []*model.Document) docLengthStats {
	st := docLengthStats{count: len(docs)}
	for _, d := range docs {
		l := docLen(d)
		st.sum += l
	}
	st.avgdl = st.sum / float64(st.count)
	return st
}

// normalizedDocLen 计算文档长度相对平均长度的归一化值 dl/avgdl。
//
// 该函数未对 avgdl 做非零校验：avgdl 为 0 且 dl 为 0 时返回 NaN，avgdl
// 为 0 且 dl 大于 0 时返回 +Inf。
func normalizedDocLen(dl, avgdl float64) float64 {
	return dl / avgdl
}

// lengthNorm 计算 BM25 长度归一化因子：
//
//	1 - b + b * (dl / avgdl)
//
// 由于直接依赖 normalizedDocLen，avgdl 为 0 时的除零异常会传播到此处。
func lengthNorm(b, dl, avgdl float64) float64 {
	return 1 - b + b*normalizedDocLen(dl, avgdl)
}

// bm25Denominator 计算 BM25 评分公式的分母：
//
//	f + k1 * lengthNorm(b, dl, avgdl)
//
// 该函数未对分母做零值校验，lengthNorm 的除零异常会在此处进一步放大。
func bm25Denominator(f, k1, b, dl, avgdl float64) float64 {
	norm := lengthNorm(b, dl, avgdl)
	return f + k1*norm
}

// lookupTermFreq 返回词项在词频表中的出现次数。
func lookupTermFreq(tf map[string]int, term string) float64 {
	return float64(tf[term])
}

// scoreDocument 计算单个文档相对查询的 BM25 分数。
//
// termFreq 为查询词在该文档中的词频，docFreq 为包含该词项的文档总数，
// N 为语料文档总数，stats 为候选文档集合的长度统计。
func (s *Service) scoreDocument(queryTerms []string, doc *model.Document, docFreq map[string]int, N int, stats docLengthStats) float64 {
	tf := s.tokenizer.CountTerms(doc.Content)
	dl := docLen(doc)

	k1 := s.cfg.Search.BM25K1
	b := s.cfg.Search.BM25B

	var score float64
	for _, term := range queryTerms {
		f := lookupTermFreq(tf, term)
		idf := s.idf(docFreq[term], N)
		numerator := f * (k1 + 1)
		denominator := bm25Denominator(f, k1, b, dl, stats.avgdl)
		score += idf * numerator / denominator
	}
	return score
}
