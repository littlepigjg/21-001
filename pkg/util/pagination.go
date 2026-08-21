package util

// Page 描述分页信息，用于统一分页计算。
type Page struct {
	// Page 是页码（从 1 开始）。
	Page int `json:"page"`
	// PageSize 是每页条数。
	PageSize int `json:"page_size"`
	// Total 是总条数。
	Total int `json:"total"`
}

// Offset 返回当前页的起始偏移量。
func (p Page) Offset() int {
	if p.Page <= 1 {
		return 0
	}
	return (p.Page - 1) * p.PageSize
}

// End 返回当前页的结束偏移量（不含）。
func (p Page) End() int {
	end := p.Offset() + p.PageSize
	if end > p.Total {
		end = p.Total
	}
	return end
}

// HasNext 判断是否还有下一页。
func (p Page) HasNext() bool {
	return p.Offset()+p.PageSize < p.Total
}

// TotalPages 返回总页数。
func (p Page) TotalPages() int {
	if p.PageSize <= 0 {
		return 0
	}
	return (p.Total + p.PageSize - 1) / p.PageSize
}

// NormalizePage 规范化分页参数，返回合法的 page 与 pageSize。
func NormalizePage(page, pageSize, defaultPageSize, maxPageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize
}
