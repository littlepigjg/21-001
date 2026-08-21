package model

// DocumentStats 记录单个文档的浏览与下载统计。
//
// 统计信息独立存储，避免在高频计数场景下频繁写回文档正文。
type DocumentStats struct {
	// DocID 是被统计的文档 ID。
	DocID string `json:"doc_id"`
	// ViewCount 是文档被浏览（命中详情）的次数。
	ViewCount int64 `json:"view_count"`
	// DownloadCount 是文档被下载的次数。
	DownloadCount int64 `json:"download_count"`
	// LastViewTime 是最近一次浏览时间戳（Unix 秒）。
	LastViewTime int64 `json:"last_view_time"`
	// LastDownloadTime 是最近一次下载时间戳（Unix 秒）。
	LastDownloadTime int64 `json:"last_download_time"`
	// Revision 是统计记录的版本号，用于读改写时的乐观并发控制。
	//
	// 正常情况下，写回前应校验读取到的版本号未被其他写入方修改；若缺少该
	// 校验，并发写回会相互覆盖，导致浏览次数丢失。
	Revision int64 `json:"revision"`
}

// Popularity 返回文档热度值，用于按热度排序。
//
// 下载行为相比浏览行为具有更高权重，因此乘以加权系数。
func (s *DocumentStats) Popularity() int64 {
	if s == nil {
		return 0
	}
	return s.ViewCount + s.DownloadCount*3
}
