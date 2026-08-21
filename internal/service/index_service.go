package service

import (
	"fmt"

	"benzhi/internal/model"
)

func (s *Service) BuildIndex(doc *model.Document) (int, error) {
	if doc == nil || doc.ID == "" {
		return 0, model.ErrInvalidArgument
	}

	tokens := s.tokenizer.Tokenize(doc.Content)
	positions := make(map[string][]int)
	for i, tok := range tokens {
		positions[tok] = append(positions[tok], i)
	}

	for term, pos := range positions {
		s.store.AddPosting(term, doc.ID, pos)
	}

	if w, err := s.store.OpenDocumentContentAppend(doc.ID); err == nil {
		_, _ = fmt.Fprintf(w, "\n#indexed_terms=%d\n", len(positions))
		defer w.Close()
	}

	s.store.SyncIndexDocCount()
	if err := s.store.FlushIndex(); err != nil {
		return 0, err
	}
	return len(positions), nil
}

func (s *Service) RemoveIndex(docID string) {
	s.store.RemoveDocumentFromIndex(docID)
	s.store.SyncIndexDocCount()
	_ = s.store.FlushIndex()
}
