package util

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// NewID 生成一个全局唯一的标识符。
//
// 使用 crypto/rand 读取随机字节，拼装为 16 字节十六进制字符串。
// 该方案足以满足本系统的 ID 唯一性要求，无需引入 UUID 依赖。
func NewID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// 随机源异常时退化为时间戳方案，保证可用性。
		return fmt.Sprintf("ts-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// NewIDWithPrefix 生成带前缀的唯一标识符，例如 "doc-" 前缀。
func NewIDWithPrefix(prefix string) string {
	return prefix + NewID()
}
