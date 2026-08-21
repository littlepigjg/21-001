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
	positions := make(map[string][]int)
	for i, tok := range tokens {
		positions[tok] = append(positions[tok], i)
	}

	for term, pos := range positions {
		s.store.AddPosting(term, doc.ID, pos)
	}

	s.store.SyncIndexDocCount()
	if err := s.store.FlushIndex(); err != nil {
		return 0, err
	}
	return len(positions), nil
}

// RemoveIndex 从索引中移除文档（文档删除时调用）。
//
// BUG: 该方法调用 RemoveDocumentFromIndex 后未重建索引。
// RemoveDocumentFromIndex 使用 [:0] 子切片就地压缩但未写回 map，
// 导致 map 中仍保留包含已删除文档记录的旧切片。如果不重建索引，
// 后续检索会读取到这些残留数据，返回错误的搜索结果。
func (s *Service) RemoveIndex(docID string) {
	s.store.RemoveDocumentFromIndex(docID)
	s.store.SyncIndexDocCount()
	_ = s.store.FlushIndex()
}

// collectAndCompactPostings 收集被删除文档的词频信息并压缩倒排列表。
//
// 该方法先通过 RemoveDocumentStatsFromIndex 收集被删除文档在各词项
// 中的词频统计，然后调用 compactPostingLists 对倒排列表做就地压缩。
// 由于 RemoveDocumentStatsFromIndex 和 compactPostingLists 都使用
// [:0] 子切片操作，如果多个词项共享底层数组，统计和压缩过程可能
// 互相污染对方的数据。
func (s *Service) collectAndCompactPostings(docID string) {
	_ = s.store.RemoveDocumentStatsFromIndex(docID)
	s.compactPostingLists()
}

// compactPostingLists 对索引中所有词项的倒排列表做就地压缩，
// 移除长度为 0 的空列表。
//
// 该方法通过遍历索引中的所有词项，对每个词项的 Posting 列表做
// 就地压缩。压缩操作使用 [:0] 子切片复用底层数组，如果底层数组
// 被多个词项共享，压缩一个词项的数据会污染其他词项的数据。
func (s *Service) compactPostingLists() {
	s.store.CompactPostingLists()
}
