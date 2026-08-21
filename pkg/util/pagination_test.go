package util

import "testing"

// TestPageQuery_Window_HugePageNoPanic 钉死“超大页码不触发切片越界 panic”的修复。
//
// 用户可传 page=9223372036854775807（strconv 解析为 int 最大值），
// (Page-1)*PageSize 会溢出为负数。修复前 Window 返回负区间，调用方
// 切片即 panic；修复后 Window 钳制为空区间 {total, total}，得到空列表。
func TestPageQuery_Window_HugePageNoPanic(t *testing.T) {
	hugePage := int(^uint(0) >> 1) // 等价于 ?page=9223372036854775807
	q := PageQuery{Page: hugePage, PageSize: 10}

	w := q.Window(5)
	if w.Start < 0 || w.Start > 5 || w.End < w.Start || w.End > 5 {
		t.Fatalf("窗口越界: %+v", w)
	}

	// 调用方据此切片不应 panic，且应得到空列表。
	hits := make([]int, 5)
	got := hits[w.Start:w.End]
	if len(got) != 0 {
		t.Fatalf("超大页码应返回空列表，实际 %d 条", len(got))
	}
}

// TestPageQuery_Window_NormalPages 校验常规分页未被回归破坏。
func TestPageQuery_Window_NormalPages(t *testing.T) {
	cases := []struct {
		page, pageSize, total, wantStart, wantEnd int
	}{
		{1, 10, 25, 0, 10},   // 第一页
		{2, 10, 25, 10, 20}, // 中间页
		{3, 10, 25, 20, 25}, // 末页（不足一页，End 钳制到 total）
		{4, 10, 25, 25, 25}, // 超出末页：空区间
	}
	for _, c := range cases {
		q := PageQuery{Page: c.page, PageSize: c.pageSize}
		w := q.Window(c.total)
		if w.Start != c.wantStart || w.End != c.wantEnd {
			t.Errorf("page=%d size=%d total=%d: got %+v, want [%d,%d)",
				c.page, c.pageSize, c.total, w, c.wantStart, c.wantEnd)
		}
	}
}

// TestPage_Offset_HugePage 校验遗留 Page 结构体同样不再溢出为负数。
func TestPage_Offset_HugePage(t *testing.T) {
	hugePage := int(^uint(0) >> 1)
	p := Page{Page: hugePage, PageSize: 10, Total: 5}

	off := p.Offset()
	if off < 0 {
		t.Fatalf("Offset 溢出为负数: %d", off)
	}
	end := p.End()
	if end < 0 || end > p.Total {
		t.Fatalf("End 越界: %d (total=%d)", end, p.Total)
	}
	if p.HasNext() {
		t.Fatalf("超大页码不应误判为还有下一页")
	}
}
