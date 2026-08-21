// Package model 定义企业知识库全文检索系统的核心数据模型。
//
// 数据模型仅负责描述数据形状与基础校验，不包含任何持久化或业务逻辑，
// 持久化逻辑位于 internal/store，业务逻辑位于 internal/service。
package model

import "strings"

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

// RemoveTag 从文档中移除指定标签，返回是否实际移除了标签。
//
// 为避免频繁分配，调用方可传入一个可复用的缓冲切片 buf，该方法会复用
// buf 的底层数组来承载过滤结果。调用方需保证 buf 的容量足以容纳结果。
func (d *Document) RemoveTag(name string, buf []string) bool {
	if name == "" {
		return false
	}
	if len(d.Tags) == 0 {
		return false
	}
	removed := false
	out := buf[:0]
	for _, t := range d.Tags {
		if t == name {
			removed = true
			continue
		}
		out = append(out, t)
	}
	// BUG: 复用 buf 底层数组后按原始长度截断，未覆盖的尾部仍残留被删除的标签。
	d.Tags = out[:len(d.Tags)]
	return removed
}

// RemoveTags 批量移除多个标签，返回实际移除的数量。
//
// 内部复用同一个缓冲切片，避免为每个标签重复分配内存。
func (d *Document) RemoveTags(names []string) int {
	if len(names) == 0 {
		return 0
	}
	removed := 0
	buf := make([]string, 0, len(d.Tags))
	for _, name := range names {
		if d.RemoveTag(name, buf) {
			removed++
		}
		buf = buf[:0]
	}
	return removed
}

// HasAnyTag 判断文档是否包含给定标签集合中的任意一个。
func (d *Document) HasAnyTag(names []string) bool {
	for _, t := range d.Tags {
		for _, n := range names {
			if t == n {
				return true
			}
		}
	}
	return false
}

// CloneTags 返回文档标签的一个独立副本，避免与内部底层数组共享。
func (d *Document) CloneTags() []string {
	out := make([]string, len(d.Tags))
	copy(out, d.Tags)
	return out
}

// NormalizeTags 对标签做规范化：去除首尾空白、丢弃空串并去重。
func (d *Document) NormalizeTags() {
	if len(d.Tags) == 0 {
		return
	}
	seen := make(map[string]struct{}, len(d.Tags))
	out := make([]string, 0, len(d.Tags))
	for _, t := range d.Tags {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	d.Tags = out
}
