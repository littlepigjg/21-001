package model

type Posting struct {
	DocID string `json:"doc_id"`

	TermFreq int `json:"tf"`

	Positions []int `json:"positions,omitempty"`
}

type PostingList struct {
	Term string `json:"term"`

	DocFreq int `json:"doc_freq"`

	Postings []Posting `json:"postings"`
}

type InvertedIndex struct {
	Version int `json:"version"`

	Terms map[string]PostingList `json:"terms"`

	DocCount int `json:"doc_count"`
}

func NewInvertedIndex() *InvertedIndex {
	return &InvertedIndex{
		Version:  1,
		Terms:    make(map[string]PostingList),
		DocCount: 0,
	}
}

func (idx *InvertedIndex) Lookup(term string) PostingList {
	if idx == nil || idx.Terms == nil {
		return PostingList{Term: term}
	}
	return idx.Terms[term]
}

func (idx *InvertedIndex) Has(term string) bool {
	if idx == nil || idx.Terms == nil {
		return false
	}
	_, ok := idx.Terms[term]
	return ok
}
