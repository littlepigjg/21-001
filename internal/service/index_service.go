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
	tokens := s.tokenizer.Tokenize(doc.Content)
	termPositions := aggregateTermPositions(tokens)

	// 逐词项构造倒排列表，并合并写入共享倒排索引。
	written := 0
	for term, positions := range termPositions {
		pl := model.PostingList{
			Term: term,
			Postings: []model.Posting{{
				DocID:     doc.ID,
				TermFreq:  len(positions),
				Positions: positions,
			}},
		}
		if s.store.MergePostingList(term, pl) {
			written++
		}
	}

	s.store.SyncIndexDocCount()
	if err := s.store.FlushIndex(); err != nil {
		return 0, err
	}
	return written, nil
}

// aggregateTermPositions 将分词结果按词项聚合出现位置。
func aggregateTermPositions(tokens []string) map[string][]int {
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
