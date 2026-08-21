package model

// Category 表示文档分类。
//
// 分类与标签的区别在于：一篇文档只能属于一个分类，但可以关联多个标签。
type Category struct {
	// ID 是分类的唯一标识。
	ID string `json:"id"`
	// Name 是分类名称，全局唯一。
	Name string `json:"name"`
	// Description 是分类的简短描述。
	Description string `json:"description"`
	// CreateTime 是分类创建时间戳（Unix 秒）。
	CreateTime int64 `json:"create_time"`
}

// Normalize 对分类做基础规范化。
func (c *Category) Normalize() {
	if c.Description == "" {
		c.Description = ""
	}
}
