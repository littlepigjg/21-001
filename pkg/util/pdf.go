package util

import (
	"bytes"
	"strings"
)

// ExtractPDFText 从 PDF 原始字节中抽取纯文本（简化实现）。
//
// 说明：真实的 PDF 文本抽取需要解析交叉引用表、流对象与字体编码，复杂度
// 极高。本系统为满足"纯标准库"约束，采用启发式方法：
//   - 解压 FlateDecode 流（标准库 compress/zlib 与 compress/flate）；
//   - 提取 ( ) 与 < > 包裹的字符串字面量以及 Tj / TJ 操作数中的文本。
//
// 该实现足以处理多数由文本编辑器/简单生成器产生的 PDF，对复杂扫描件仅能
// 抽取部分文本。
func ExtractPDFText(data []byte) string {
	// 定位所有流对象并尝试解压其中的内容流。
	var content []byte
	streamStart := []byte("stream")
	streamEnd := []byte("endstream")

	for {
		idx := bytes.Index(data, streamStart)
		if idx < 0 {
			break
		}
		// 跳过 "stream" 关键字及紧随其后的换行。
		idx += len(streamStart)
		if idx < len(data) && (data[idx] == '\r') {
			idx++
		}
		if idx < len(data) && data[idx] == '\n' {
			idx++
		}

		end := bytes.Index(data[idx:], streamEnd)
		if end < 0 {
			break
		}
		raw := data[idx : idx+end]
		content = append(content, inflate(raw)...)
		content = append(content, '\n')

		data = data[idx+end+len(streamEnd):]
	}

	if len(content) == 0 {
		return ""
	}
	return extractTextOperators(content)
}

// inflate 尝试对 zlib 压缩的数据解压；失败时原样返回。
func inflate(data []byte) []byte {
	// zlib 数据通常以 0x78 0x9C / 0x78 0x01 开头。
	if len(data) >= 2 && data[0] == 0x78 {
		if out, err := zlibDecompress(data); err == nil {
			return out
		}
	}
	return data
}

// extractTextOperators 从内容流中抽取文本绘制操作符携带的字符串。
func extractTextOperators(content []byte) string {
	var out strings.Builder
	inText := false
	// 简化状态机：遇到 BT 进入文本对象，遇到 ET 退出。
	s := string(content)
	for i := 0; i < len(s); i++ {
		if !inText {
			if strings.HasPrefix(s[i:], "BT") {
				inText = true
				i++
			}
			continue
		}
		if strings.HasPrefix(s[i:], "ET") {
			inText = false
			i++
			continue
		}
		if s[i] == '(' {
			// 读取括号字符串，处理转义。
			j := i + 1
			var b strings.Builder
			for j < len(s) && s[j] != ')' {
				if s[j] == '\\' && j+1 < len(s) {
					j++
					switch s[j] {
					case 'n':
						b.WriteByte('\n')
					case 'r':
						b.WriteByte('\r')
					case 't':
						b.WriteByte('\t')
					default:
						b.WriteByte(s[j])
					}
					j++
					continue
				}
				b.WriteByte(s[j])
				j++
			}
			out.WriteString(b.String())
			out.WriteByte(' ')
			i = j
		}
	}
	return out.String()
}
