package model

// Tag 表示一个知识库标签。
//
// 标签用于对文档进行多维度的组织与过滤，同一文档可以关联多个标签。
type Tag struct {
	// ID 是标签的唯一标识。
	ID string `json:"id"`
	// Name 是标签名称，全局唯一。
	Name string `json:"name"`
	// Count 是当前关联该标签的文档数量（冗余计数，用于展示）。
	Count int `json:"count"`
	// CreateTime 是标签创建时间戳（Unix 秒）。
	CreateTime int64 `json:"create_time"`
}

// Normalize 对标签做基础规范化。
func (t *Tag) Normalize() {
	if t.Count < 0 {
		t.Count = 0
	}
}
