package util

import "strings"

// SplitAndTrim 按逗号/空白切分字符串并去除空项。
//
// 常用于解析请求参数中的标签列表、分类列表等。
func SplitAndTrim(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	fields := strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == '，' || r == ' ' || r == '\t'
	})
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f != "" {
			out = append(out, f)
		}
	}
	return out
}

// Truncate 将字符串截断到 maxLen，超出部分以省略号结尾。
func Truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

// ContainsString 判断字符串切片是否包含目标字符串。
func ContainsString(list []string, target string) bool {
	for _, s := range list {
		if s == target {
			return true
		}
	}
	return false
}
