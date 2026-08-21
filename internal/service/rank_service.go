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
