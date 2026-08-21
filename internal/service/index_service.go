package service

import (
	"benzhi/internal/model"
)

// BuildIndex 为文档构建倒排索引，返回本次新增的词项数量。
//
// 该函数对正文分词后，按词项聚合出现位置，并写入底层存储的倒排索引。
func (s *Service) BuildIndex(doc *model.Document) (int, error) {
	if doc == nil || doc.ID == "" {
		return 0, model.ErrInvalidArgument
	}

	// 分词并记录每个词项的出现位置。
	positions := s.buildPositions(doc.Content)

	for term, pos := range positions {
		// 缺陷：直接把位置切片以引用方式交给 store，未做防御性拷贝；
		// 一旦 store 侧复用共享缓冲存储，这些切片会被后续 append 污染。
		s.store.AddPosting(term, doc.ID, pos)
	}

	s.store.SyncIndexDocCount()
	if err := s.store.FlushIndex(); err != nil {
		return 0, err
	}
	return len(positions), nil
}

// buildPositions 对正文分词，并聚合每个词项的出现位置。
func (s *Service) buildPositions(content string) map[string][]int {
	tokens := s.tokenizer.Tokenize(content)
	positions := make(map[string][]int)
	for i, tok := range tokens {
		positions[tok] = append(positions[tok], i)
	}
	return positions
}

// RemoveIndex 从索引中移除文档（文档删除时调用）。
func (s *Service) RemoveIndex(docID string) {
	s.store.RemoveDocumentFromIndex(docID)
	s.store.SyncIndexDocCount()
	_ = s.store.FlushIndex()
}
