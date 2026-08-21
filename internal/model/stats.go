package model

type DocumentStats struct {
	DocID string `json:"doc_id"`

	ViewCount int64 `json:"view_count"`

	DownloadCount int64 `json:"download_count"`

	LastViewTime int64 `json:"last_view_time"`

	LastDownloadTime int64 `json:"last_download_time"`
}

func (s *DocumentStats) Popularity() int64 {
	if s == nil {
		return 0
	}
	return s.ViewCount + s.DownloadCount*3
}
