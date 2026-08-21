package service

// OverviewMetrics 描述系统的概览统计信息。
type OverviewMetrics struct {
	// DocumentCount 是文档总数。
	DocumentCount int `json:"document_count"`
	// TagCount 是标签总数。
	TagCount int `json:"tag_count"`
	// CategoryCount 是分类总数。
	CategoryCount int `json:"category_count"`
	// IndexTermCount 是倒排索引词项总数。
	IndexTermCount int `json:"index_term_count"`
	// TotalViews 是所有文档浏览次数之和。
	TotalViews int64 `json:"total_views"`
	// TotalDownloads 是所有文档下载次数之和。
	TotalDownloads int64 `json:"total_downloads"`
}

// Overview 返回系统的概览统计信息。
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
