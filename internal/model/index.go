package model

// Posting 表示某个词项在单个文档中的出现记录。
type Posting struct {
	// DocID 是包含该词项的文档 ID。
	DocID string `json:"doc_id"`
	// TermFreq 是词项在该文档中出现的次数（词频 TF）。
	TermFreq int `json:"tf"`
	// Positions 是词项在该文档中出现的所有位置（用于短语/邻近检索扩展）。
	Positions []int `json:"positions,omitempty"`
}

// PostingList 表示某个词项对应的倒排列表。
type PostingList struct {
	// Term 是词项本身。
	Term string `json:"term"`
	// DocFreq 是包含该词项的文档总数。
	DocFreq int `json:"doc_freq"`
	// Postings 是按文档组织的出现记录列表。
	Postings []Posting `json:"postings"`
}

// InvertedIndex 表示完整的倒排索引结构。
//
// 索引以 JSON 形式持久化，检索时直接加载到内存中的 map 以加速查询。
type InvertedIndex struct {
	// Version 是索引结构版本号，便于未来迁移。
	Version int `json:"version"`
	// Terms 是词项到倒排列表的映射。
	Terms map[string]PostingList `json:"terms"`
	// DocCount 是已建索引的文档总数。
	DocCount int `json:"doc_count"`
}

// NewInvertedIndex 创建一个空的倒排索引。
func NewInvertedIndex() *InvertedIndex {
	return &InvertedIndex{
		Version:  1,
		Terms:    make(map[string]PostingList),
		DocCount: 0,
	}
}

// Lookup 返回指定词项对应的倒排列表；若不存在则返回零值列表。
func (idx *InvertedIndex) Lookup(term string) PostingList {
	if idx == nil || idx.Terms == nil {
		return PostingList{Term: term}
	}
	return idx.Terms[term]
}

// Has 判断词项是否存在于索引中。
func (idx *InvertedIndex) Has(term string) bool {
	if idx == nil || idx.Terms == nil {
		return false
	}
	_, ok := idx.Terms[term]
	return ok
}
