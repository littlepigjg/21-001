package util

import (
	"bytes"
	"strings"
)

func ExtractPDFText(data []byte) string {

	var content []byte
	streamStart := []byte("stream")
	streamEnd := []byte("endstream")

	for {
		idx := bytes.Index(data, streamStart)
		if idx < 0 {
			break
		}

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

func inflate(data []byte) []byte {

	if len(data) >= 2 && data[0] == 0x78 {
		if out, err := zlibDecompress(data); err == nil {
			return out
		}
	}
	return data
}

func extractTextOperators(content []byte) string {
	var out strings.Builder
	inText := false

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
