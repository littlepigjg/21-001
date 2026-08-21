package util

import (
	"bytes"
	"compress/zlib"
	"io"
)

// zlibDecompress 解压 zlib 压缩数据。
func zlibDecompress(data []byte) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}
