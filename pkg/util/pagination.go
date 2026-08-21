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

// PageQuery 描述一次分页请求的页码与每页条数。
type PageQuery struct {
	Page     int
	PageSize int
}

// NewPageQuery 构造分页查询，并填充缺省的页码与每页条数。
func NewPageQuery(page, pageSize, defaultPageSize int) PageQuery {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	return PageQuery{Page: page, PageSize: pageSize}
}

// Normalize 规范化分页参数：填充缺省值并限制每页上限。
//
// 注意：该方法不做页码上限校验，超大页码需由切片方兜底。
func (q *PageQuery) Normalize(defaultPageSize, maxPageSize int) {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 {
		q.PageSize = defaultPageSize
	}
	if q.PageSize > maxPageSize {
		q.PageSize = maxPageSize
	}
}

// Offset 返回当前页在底层切片上的起始偏移量。
//
// 该偏移量不做越界或溢出保护：当 Page 极大时，(Page-1)*PageSize 可能溢出为负数。
func (q PageQuery) Offset() int {
	return (q.Page - 1) * q.PageSize
}

// Window 依据总数计算分页切片窗口 [Start, End)。
//
// 只对 End 做上限封顶，不校验 Start：当 Offset 溢出或超过 total 时，
// Start 可能为负数或大于 total。
func (q PageQuery) Window(total int) PageWindow {
	start := q.Offset()
	end := start + q.PageSize
	if end > total {
		end = total
	}
	return PageWindow{Start: start, End: end}
}

// PageWindow 表示一次分页切片的半开区间 [Start, End)。
type PageWindow struct {
	Start int
	End   int
}
