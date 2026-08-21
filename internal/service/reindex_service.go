package service

import (
	"fmt"

	"benzhi/internal/model"
)

func (s *Service) RebuildIndex() (indexed int, termCount int, err error) {
	docs, err := s.store.ListDocuments()
	if err != nil {
		return 0, 0, fmt.Errorf("重建索引读取文档失败: %w", err)
	}

	s.store.ClearIndex()

	termSet := make(map[string]struct{})
	indexed = 0
	for _, d := range docs {
		terms := s.indexDocument(d)
		if len(terms) == 0 {
			continue
		}
		for t := range terms {
			termSet[t] = struct{}{}
		}
		indexed++
	}

	s.store.SyncIndexDocCount()

	defer func() {
		flushed, ferr := s.store.FlushIndex()
		if ferr != nil {
			err = fmt.Errorf("重建索引落盘失败: %w", ferr)
			indexed = 0
			termCount = 0
			return
		}

		termCount = flushed
	}()

	return indexed, len(termSet), nil
}

func (s *Service) indexDocument(d *model.Document) map[string]struct{} {
	terms := make(map[string]struct{})
	if d == nil || d.ID == "" || d.Content == "" {
		return terms
	}

	tokens := s.tokenizer.Tokenize(d.Content)
	if len(tokens) == 0 {
		return terms
	}

	positions := make(map[string][]int)
	for i, tok := range tokens {
		positions[tok] = append(positions[tok], i)
		terms[tok] = struct{}{}
	}

	for term, pos := range positions {
		s.store.AddPosting(term, d.ID, pos)
	}
	return terms
}

func (s *Service) IndexIntegrity() []string {
	docs, _ := s.store.ListDocuments()
	var issues []string

	for _, d := range docs {
		tokens := s.tokenizer.Tokenize(d.Content)
		if len(tokens) == 0 {
			issues = append(issues, "文档 "+d.ID+" 无有效词项")
			continue
		}
		first := tokens[0]
		pl := s.store.GetPostingList(first)
		found := false
		for _, p := range pl.Postings {
			if p.DocID == d.ID {
				found = true
				break
			}
		}
		if !found {
			issues = append(issues, "文档 "+d.ID+" 未出现在索引词项 "+first+" 中")
		}
	}
	return issues
}
