package service

type OverviewMetrics struct {
	DocumentCount int `json:"document_count"`

	TagCount int `json:"tag_count"`

	CategoryCount int `json:"category_count"`

	IndexTermCount int `json:"index_term_count"`

	TotalViews int64 `json:"total_views"`

	TotalDownloads int64 `json:"total_downloads"`
}

func (s *Service) Overview() OverviewMetrics {
	docs, _ := s.store.ListDocuments()
	tags, _ := s.store.ListTags()
	cats, _ := s.store.ListCategories()
	stats := s.store.ListStats()

	m := OverviewMetrics{
		DocumentCount:  len(docs),
		TagCount:       len(tags),
		CategoryCount:  len(cats),
		IndexTermCount: s.store.IndexTermCount(),
	}
	for _, st := range stats {
		m.TotalViews += st.ViewCount
		m.TotalDownloads += st.DownloadCount
	}
	return m
}
