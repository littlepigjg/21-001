package store

import (
	"benzhi/internal/model"
	"benzhi/pkg/util"
)

func (s *Store) AddPosting(term, docID string, positions []int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.addPostingLocked(term, docID, positions)
}

func (s *Store) addPostingLocked(term, docID string, positions []int) {
	if s.index.Terms == nil {
		s.index.Terms = make(map[string]model.PostingList)
	}
	pl, ok := s.index.Terms[term]
	if !ok {
		pl = model.PostingList{Term: term, Postings: []model.Posting{}}
	}

	found := false
	for i := range pl.Postings {
		if pl.Postings[i].DocID == docID {
			pl.Postings[i].TermFreq = len(positions)
			pl.Postings[i].Positions = positions
			found = true
			break
		}
	}
	if !found {
		pl.Postings = append(pl.Postings, model.Posting{
			DocID:     docID,
			TermFreq:  len(positions),
			Positions: positions,
		})
	}
	pl.DocFreq = len(pl.Postings)
	s.index.Terms[term] = pl
}

func (s *Store) GetPostingList(term string) model.PostingList {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.index.Lookup(term)
}

func (s *Store) RemoveDocumentFromIndex(docID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for term, pl := range s.index.Terms {
		out := pl.Postings[:0]
		for _, p := range pl.Postings {
			if p.DocID != docID {
				out = append(out, p)
			}
		}
		if len(out) == 0 {
			delete(s.index.Terms, term)
			continue
		}
		pl.Postings = out
		pl.DocFreq = len(out)
		s.index.Terms[term] = pl
	}
}

func (s *Store) IndexDocCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.index.DocCount
}

func (s *Store) SetIndexDocCount(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.index.DocCount = n
}

func (s *Store) IndexTermCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.index.Terms)
}

func (s *Store) SyncIndexDocCount() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.index.DocCount = len(s.documents)
}

func (s *Store) ClearIndex() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.index.Terms = make(map[string]model.PostingList)
	s.index.DocCount = 0
}

func (s *Store) FlushIndex() (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := util.SaveJSON(s.path(s.cfg.IndexFile), s.index); err != nil {
		return 0, err
	}
	return s.flushedTermCount(), nil
}

func (s *Store) flushedTermCount() int {
	return s.index.DocCount
}
