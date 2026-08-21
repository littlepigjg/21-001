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
//
// 含溢出保护：当 (Page-1)*PageSize 会溢出为负数时，返回 maxInt 作为
// “远超末页”的哨兵，交由 End 钳制到 total，从而避免切片越界 panic。
func (p Page) Offset() int {
	if p.Page <= 1 || p.PageSize <= 0 {
		return 0
	}
	if p.Page-1 > maxInt/p.PageSize {
		return maxInt
	}
	return (p.Page - 1) * p.PageSize
}

// End 返回当前页的结束偏移量（不含）。
//
// 对 [0, Total] 钳制：当 Offset 溢出或超过 Total 时返回 Total，
// 保证切片区间始终合法。
func (p Page) End() int {
	end := p.Offset() + p.PageSize
	if end > p.Total || end < 0 {
		end = p.Total
	}
	return end
}

// HasNext 判断是否还有下一页。
//
// 借助受保护的 Offset，超大页码下不会因溢出而误判为“还有下一页”。
func (p Page) HasNext() bool {
	return p.Offset() < p.Total && p.Offset()+p.PageSize < p.Total
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
// 该方法不做页码上限校验：超大页码的越界与溢出保护统一由
// Offset/Window 兜底，调用方据此切片即可，无需再次钳制。
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

// maxInt 是当前平台 int 的最大值，用作“远超末页”的哨兵偏移量。
const maxInt = int(^uint(0) >> 1)

// Offset 返回当前页在底层切片上的起始偏移量。
//
// 含溢出保护：当 (Page-1)*PageSize 会溢出为负数时，返回 maxInt 作为
// “远超末页”的哨兵，交由 Window 钳制到空区间，从而避免切片越界 panic。
// 乘法前先用除法判定是否溢出，避免先溢出再比较。
func (q PageQuery) Offset() int {
	if q.Page <= 1 || q.PageSize <= 0 {
		return 0
	}
	if q.Page-1 > maxInt/q.PageSize {
		return maxInt
	}
	return (q.Page - 1) * q.PageSize
}

// Window 依据总数计算分页切片窗口 [Start, End)。
//
// 对 Start 与 End 均做 [0, total] 钳制，保证返回的区间始终落在底层
// 切片范围内：当 Page 极大导致偏移溢出或超过 total 时，返回空窗口
// {Start: total, End: total}，调用方得到空列表而非 panic。
func (q PageQuery) Window(total int) PageWindow {
	start := q.Offset()
	if start < 0 || start > total {
		start = total
	}
	end := start + q.PageSize
	if end > total || end < start {
		end = total
	}
	return PageWindow{Start: start, End: end}
}

// PageWindow 表示一次分页切片的半开区间 [Start, End)。
type PageWindow struct {
	Start int
	End   int
}
