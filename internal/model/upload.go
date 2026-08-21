package model

type UploadResult struct {
	Document Document `json:"document"`

	IndexedTerms int `json:"indexed_terms"`

	Duplicated bool `json:"duplicated"`
}
