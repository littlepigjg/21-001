package store

import (
	"benzhi/internal/model"
	"benzhi/pkg/util"
)

func (s *Store) EnsureStats(docID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.stats[docID]; !ok {
		s.stats[docID] = &model.DocumentStats{DocID: docID}
	}
}

func (s *Store) GetStats(docID string) model.DocumentStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if st, ok := s.stats[docID]; ok {
		return *st
	}
	return model.DocumentStats{DocID: docID}
}

func (s *Store) IncrementView(docID string) model.DocumentStats {
	s.mu.Lock()
	defer s.mu.Unlock()

	st, ok := s.stats[docID]
	if !ok {
		st = &model.DocumentStats{DocID: docID}
		s.stats[docID] = st
	}
	st.ViewCount++
	st.LastViewTime = util.Now()
	s.persistLocked()
	return *st
}

func (s *Store) IncrementDownload(docID string) model.DocumentStats {
	s.mu.Lock()
	defer s.mu.Unlock()

	st, ok := s.stats[docID]
	if !ok {
		st = &model.DocumentStats{DocID: docID}
		s.stats[docID] = st
	}
	st.DownloadCount++
	st.LastDownloadTime = util.Now()
	s.persistLocked()
	return *st
}

func (s *Store) ListStats() []model.DocumentStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]model.DocumentStats, 0, len(s.stats))
	for _, st := range s.stats {
		out = append(out, *st)
	}
	return out
}

func (s *Store) DeleteStats(docID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.stats, docID)
}
