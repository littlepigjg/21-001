package util

import "time"

// Now 返回当前 Unix 秒时间戳。
func Now() int64 {
	return time.Now().Unix()
}

// FormatTime 将 Unix 秒时间戳格式化为可读字符串。
func FormatTime(unix int64) string {
	if unix <= 0 {
		return ""
	}
	return time.Unix(unix, 0).Format("2006-01-02 15:04:05")
}

// SinceUnix 计算从指定 Unix 秒时间戳到现在的时长。
func SinceUnix(unix int64) time.Duration {
	return time.Since(time.Unix(unix, 0))
}
