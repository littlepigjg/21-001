package util

type Page struct {
	Page int `json:"page"`

	PageSize int `json:"page_size"`

	Total int `json:"total"`
}

func (p Page) Offset() int {
	if p.Page <= 1 {
		return 0
	}
	return (p.Page - 1) * p.PageSize
}

func (p Page) End() int {
	end := p.Offset() + p.PageSize
	if end > p.Total {
		end = p.Total
	}
	return end
}

func (p Page) HasNext() bool {
	return p.Offset()+p.PageSize < p.Total
}

func (p Page) TotalPages() int {
	if p.PageSize <= 0 {
		return 0
	}
	return (p.Total + p.PageSize - 1) / p.PageSize
}

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
