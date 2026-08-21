package service

// RebuildIndex 清空现有倒排索引，并基于全部文档重新构建。
//
// 该操作用于在索引损坏或结构升级后重建索引。返回重建的文档数与词项数。
func (s *Service) RebuildIndex() (int, int, error) {
	docs, err := s.store.ListDocuments()
	if err != nil {
		return 0, 0, err
	}

	// 清空索引后重建。
	s.store.ClearIndex()

	termSet := make(map[string]struct{})
	indexed := 0
	for _, d := range docs {
		if d.Content == "" {
			continue
		}
		tokens := s.tokenizer.Tokenize(d.Content)
		positions := make(map[string][]int)
		for i, tok := range tokens {
			positions[tok] = append(positions[tok], i)
			termSet[tok] = struct{}{}
		}
		for term, pos := range positions {
			s.store.AddPosting(term, d.ID, pos)
		}
		indexed++
	}

	s.store.SyncIndexDocCount()
	if err := s.store.FlushIndex(); err != nil {
		return 0, 0, err
	}
	return indexed, len(termSet), nil
}

// IndexIntegrity 检查索引与文档表之间的一致性，返回不一致项描述。
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
