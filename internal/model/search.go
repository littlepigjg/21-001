package model

// SearchRequest 描述一次全文检索请求。
type SearchRequest struct {
	// Query 是检索关键词，支持空格分隔的多个关键词。
	Query string `json:"query"`
	// Phrase 是可选短语，非空时要求查询词项在文档中按顺序连续出现。
	Phrase string `json:"phrase,omitempty"`
	// Category 是可选分类过滤条件，为空表示不过滤。
	Category string `json:"category"`
	// Tags 是可选标签过滤条件，为空表示不过滤。
	Tags []string `json:"tags"`
	// Page 是页码，从 1 开始。
	Page int `json:"page"`
	// PageSize 是每页条数，受配置上限约束。
	PageSize int `json:"page_size"`
	// SortBy 指定排序方式：relevance（相关度）/ hot（热度）/ time（时间）。
	SortBy string `json:"sort_by"`
}

// Normalize 对检索请求做规范化，填充缺省分页参数。
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

// SearchHit 表示一条检索命中结果。
type SearchHit struct {
	// Document 是命中的文档元数据（不含正文，节省带宽）。
	Document Document `json:"document"`
	// Score 是相关度评分。
	Score float64 `json:"score"`
	// ViewCount 是文档浏览次数。
	ViewCount int64 `json:"view_count"`
	// DownloadCount 是文档下载次数。
	DownloadCount int64 `json:"download_count"`
}

// SearchResult 表示一次检索的完整返回结果。
type SearchResult struct {
	// Query 是回显的检索关键词。
	Query string `json:"query"`
	// Total 是命中的文档总数。
	Total int `json:"total"`
	// Page 是当前页码。
	Page int `json:"page"`
	// PageSize 是当前页大小。
	PageSize int `json:"page_size"`
	// Hits 是当前页的命中列表。
	Hits []SearchHit `json:"hits"`
	// TookMS 是检索耗时（毫秒）。
	TookMS int64 `json:"took_ms"`
}
