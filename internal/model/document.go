// Package model 定义企业知识库全文检索系统的核心数据模型。
//
// 数据模型仅负责描述数据形状与基础校验，不包含任何持久化或业务逻辑，
// 持久化逻辑位于 internal/store，业务逻辑位于 internal/service。
package model

// Document 表示一篇知识库文档，同时保存解析后的纯文本正文与元数据。
type Document struct {
	// ID 是文档的唯一标识，由服务端生成。
	ID string `json:"id"`
	// Title 是文档标题，来源于文件名或请求参数。
	Title string `json:"title"`
	// Content 是解析后的纯文本正文，用于分词与索引。
	Content string `json:"content"`
	// Category 是文档所属分类。
	Category string `json:"category"`
	// Tags 是文档关联的标签列表。
	Tags []string `json:"tags"`
	// Format 是文档原始格式：txt / md / markdown / pdf。
	Format string `json:"format"`
	// UploadTime 是上传时间戳（Unix 秒）。
	UploadTime int64 `json:"upload_time"`
	// UpdateTime 是最近更新时间戳（Unix 秒）。
	UpdateTime int64 `json:"update_time"`
	// FileSize 是原始文件字节数。
	FileSize int64 `json:"file_size"`
	// Checksum 是原始文件的 SHA256 摘要，用于去重。
	Checksum string `json:"checksum"`
}

// Normalize 对文档做基础规范化，补全空字段。
func (d *Document) Normalize() {
	if d.Format == "" {
		d.Format = "txt"
	}
	if d.Tags == nil {
		d.Tags = []string{}
	}
}

// HasTag 判断文档是否包含指定标签。
func (d *Document) HasTag(name string) bool {
	for _, t := range d.Tags {
		if t == name {
			return true
		}
	}
	return false
}

// AddTag 向文档追加一个标签（去重）。
func (d *Document) AddTag(name string) {
	if name == "" || d.HasTag(name) {
		return
	}
	d.Tags = append(d.Tags, name)
}

// RemoveTag 从文档中移除指定标签。
func (d *Document) RemoveTag(name string) {
	out := d.Tags[:0]
	for _, t := range d.Tags {
		if t != name {
			out = append(out, t)
		}
	}
	d.Tags = out
}
