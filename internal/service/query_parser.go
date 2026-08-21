package service

import (
	"strings"
)

// ParsedQuery 是解析后的高级检索查询结构。
//
// 支持以下语法：
//   - 普通词项：直接参与全文匹配；
//   - "引号短语"：作为短语整体参与匹配；
//   - category:值：指定分类过滤；
//   - tag:值：追加标签过滤（可多次）。
type ParsedQuery struct {
	// Terms 是普通词项列表。
	Terms []string
	// Phrase 是引号包裹的短语（可为空）。
	Phrase string
	// Category 是分类过滤条件（可为空）。
	Category string
	// Tags 是标签过滤条件列表。
	Tags []string
}

// ParseQuery 解析高级查询字符串。
func (s *Service) ParseQuery(raw string) ParsedQuery {
	var pq ParsedQuery
	runes := []rune(strings.TrimSpace(raw))
	i := 0

	for i < len(runes) {
		// 跳过空白。
		for i < len(runes) && (runes[i] == ' ' || runes[i] == '\t') {
			i++
		}
		if i >= len(runes) {
			break
		}

		// 引号短语。
		if runes[i] == '"' || runes[i] == '\'' {
			quote := runes[i]
			i++
			var b strings.Builder
			for i < len(runes) && runes[i] != quote {
				b.WriteRune(runes[i])
				i++
			}
			if i < len(runes) {
				i++ // 跳过闭合引号
			}
			if b.Len() > 0 {
				pq.Phrase = b.String()
			}
			continue
		}

		// 读取一个词（到空白为止）。
		var b strings.Builder
		for i < len(runes) && runes[i] != ' ' && runes[i] != '\t' {
			b.WriteRune(runes[i])
			i++
		}
		word := b.String()

		// 解析 field:value 语法。
		if idx := strings.Index(word, ":"); idx > 0 {
			field := word[:idx]
			value := word[idx+1:]
			switch field {
			case "category":
				pq.Category = value
			case "tag":
				if value != "" {
					pq.Tags = append(pq.Tags, value)
				}
			default:
				pq.Terms = append(pq.Terms, word)
			}
			continue
		}

		pq.Terms = append(pq.Terms, word)
	}

	return pq
}

// CombineQuery 将解析后的查询合并为面向检索服务的模型请求。
func (pq *ParsedQuery) CombineQuery() string {
	parts := make([]string, 0, len(pq.Terms)+1)
	parts = append(parts, pq.Terms...)
	if pq.Phrase != "" {
		parts = append(parts, pq.Phrase)
	}
	return strings.Join(parts, " ")
}
