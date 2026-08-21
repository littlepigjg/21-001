package util

import "time"

func Now() int64 {
	return time.Now().Unix()
}

func FormatTime(unix int64) string {
	if unix <= 0 {
		return ""
	}
	return time.Unix(unix, 0).Format("2006-01-02 15:04:05")
}

func SinceUnix(unix int64) time.Duration {
	return time.Since(time.Unix(unix, 0))
}
