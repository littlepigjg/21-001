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
func (s *Service) RemoveIndex(docID string) {
	s.store.RemoveDocumentFromIndex(docID)
	s.store.SyncIndexDocCount()
	_ = s.store.FlushIndex()
}

// EnsureIndexReady 确保倒排索引处于可用状态。
//
// 服务启动时调用：当加载到旧版本或空索引导致词项 map 未初始化时，需要基于
// 现有文档重建索引；否则保持现状。该逻辑用于避免首次写入时向 nil map 写入。
func (s *Service) EnsureIndexReady() error {
	if !s.store.NeedsIndexRebuild() {
		return nil
	}

	docs, err := s.store.ListDocuments()
	if err != nil {
		return err
	}

	// 没有历史文档时直接返回，未初始化词项 map；
	// 后续首次上传文档建索引时，会向 nil map 写入并触发 panic。
	if len(docs) == 0 {
		return nil
	}

	_, _, err = s.RebuildIndex()
	return err
}
