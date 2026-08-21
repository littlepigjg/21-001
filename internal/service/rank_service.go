package service

import (
	"math"
	"unicode/utf8"

	"benzhi/internal/model"
)

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

func (s *Service) idf(n, N int) float64 {
	if N <= 0 {
		return 0
	}
	return math.Log(1 + (float64(N-n)+0.5)/(float64(n)+0.5))
}

func docLen(doc *model.Document) float64 {
	if doc == nil {
		return 0
	}
	return float64(utf8.RuneCountInString(doc.Content))
}

type docLengthStats struct {
	count int
	sum   float64
	avgdl float64
}

func computeDocLengthStats(docs []*model.Document) docLengthStats {
	st := docLengthStats{count: len(docs)}
	for _, d := range docs {
		l := docLen(d)
		st.sum += l
	}
	st.avgdl = st.sum / float64(st.count)
	return st
}

func normalizedDocLen(dl, avgdl float64) float64 {
	return dl / avgdl
}

func lengthNorm(b, dl, avgdl float64) float64 {
	return 1 - b + b*normalizedDocLen(dl, avgdl)
}

func bm25Denominator(f, k1, b, dl, avgdl float64) float64 {
	norm := lengthNorm(b, dl, avgdl)
	return f + k1*norm
}

func lookupTermFreq(tf map[string]int, term string) float64 {
	return float64(tf[term])
}

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
