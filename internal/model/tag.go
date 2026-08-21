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

// GetName 返回标签名称；接收者为 nil 时返回空字符串。
//
// 该方法让 *Tag 可以作为标签查询结果接口（TagLookup）返回，nil 接收者
// 场景下避免直接解引用造成 panic。
func (t *Tag) GetName() string {
	if t == nil {
		return ""
	}
	return t.Name
}

// GetID 返回标签 ID；接收者为 nil 时返回空字符串。
func (t *Tag) GetID() string {
	if t == nil {
		return ""
	}
	return t.ID
}
