package model

type SearchRequest struct {
	Query string `json:"query"`

	Category string `json:"category"`

	Tags []string `json:"tags"`

	Page int `json:"page"`

	PageSize int `json:"page_size"`

	SortBy string `json:"sort_by"`
}

func (r *SearchRequest) Normalize(defaultPageSize, maxPageSize int) {
	if r.Page <= 0 {
		r.Page = 1
	}
	if r.PageSize <= 0 {
		r.PageSize = defaultPageSize
	}
	if r.PageSize > maxPageSize {
		r.PageSize = maxPageSize
	}
	if r.SortBy == "" {
		r.SortBy = "relevance"
	}
	if r.Tags == nil {
		r.Tags = []string{}
	}
}

type SearchHit struct {
	Document Document `json:"document"`

	Score float64 `json:"score"`

	ViewCount int64 `json:"view_count"`

	DownloadCount int64 `json:"download_count"`
}

type SearchResult struct {
	Query string `json:"query"`

	Total int `json:"total"`

	Page int `json:"page"`

	PageSize int `json:"page_size"`

	Hits []SearchHit `json:"hits"`

	TookMS int64 `json:"took_ms"`
}
