// Package util 提供跨模块复用的基础工具函数。
package util

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
)

// SHA256Hex 计算给定字节数据的 SHA256 摘要并返回十六进制字符串。
func SHA256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// SHA256Reader 计算读取器内容的 SHA256 摘要。
func SHA256Reader(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
