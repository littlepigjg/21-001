package tokenizer

// ExtractPhrases 提取文本中长度在 [min, max] 区间的连续 n-gram 短语。
//
// n-gram 短语可用于扩展检索召回或构建更细粒度的索引。
func (t *Tokenizer) ExtractPhrases(text string, min, max int) []string {
	tokens := t.Tokenize(text)
	if min < 1 {
		min = 1
	}
	if max < min {
		max = min
	}

	var phrases []string
	for n := min; n <= max; n++ {
		for i := 0; i+n <= len(tokens); i++ {
			phrase := ""
			for j := 0; j < n; j++ {
				if j > 0 {
					phrase += " "
				}
				phrase += tokens[i+j]
			}
			phrases = append(phrases, phrase)
		}
	}
	return phrases
}

// MatchPhrase 判断词项序列 tokens 中是否包含短语 phrase 的连续出现。
//
// 该函数使用朴素的滑动窗口匹配，适用于短词项序列。
func MatchPhrase(tokens, phrase []string) bool {
	if len(phrase) == 0 {
		return false
	}
	if len(tokens) < len(phrase) {
		return false
	}

	for i := 0; i+len(phrase) <= len(tokens); i++ {
		match := true
		for j := range phrase {
			if tokens[i+j] != phrase[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
