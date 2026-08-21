package util

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

func NewID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {

		return fmt.Sprintf("ts-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

func NewIDWithPrefix(prefix string) string {
	return prefix + NewID()
}
