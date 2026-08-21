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
	positions := s.tokenizePositions(doc.Content)
	if len(positions) == 0 {
		return 0, nil
	}

	// 开启索引构建事务：先将词项写入内存倒排索引。
	s.store.BeginIndexBuild(doc.ID, positions)

	// 提交事务：同步文档计数并落盘。
	if err := s.store.CommitIndexBuild(); err != nil {
		// 落盘失败：回滚内存倒排索引词项并重算 DocCount，使其恢复到本次构建前
		// 的状态；持久化的失败状态由上层（UploadDocument 的回滚）一并处理。
		s.store.RollbackIndexBuild(doc.ID)
		return 0, err
	}
	return len(positions), nil
}

// tokenizePositions 对正文分词并聚合每个词项的出现位置。
func (s *Service) tokenizePositions(content string) map[string][]int {
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
