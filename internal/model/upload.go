package model

// UploadResult 描述一次文档上传处理的结果。
type UploadResult struct {
	// Document 是上传成功后的文档元数据。
	Document Document `json:"document"`
	// IndexedTerms 是本次为该文档新增的索引词项数量。
	IndexedTerms int `json:"indexed_terms"`
	// Duplicated 表示该文件是否因摘要相同而被判定为重复上传。
	Duplicated bool `json:"duplicated"`
}
