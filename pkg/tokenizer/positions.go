package tokenizer

// Token 表示一个带流内位置信息的词项。
type Token struct {
	// Text 是词项文本。
	Text string
	// Position 是词项在分词结果序列中的下标。
	Position int
}

// TokenizeWithPosition 返回带流内位置信息的词项列表。
//
// 与 Tokenize 的区别在于额外携带了词项下标，便于短语/邻近检索。
func (t *Tokenizer) TokenizeWithPosition(text string) []Token {
	tokens := t.Tokenize(text)
	out := make([]Token, 0, len(tokens))
	for i, tok := range tokens {
		out = append(out, Token{Text: tok, Position: i})
	}
	return out
}

// TokenStream 将词项列表转换为带位置的 Token 切片。
func TokenStream(tokens []string) []Token {
	out := make([]Token, 0, len(tokens))
	for i, tok := range tokens {
		out = append(out, Token{Text: tok, Position: i})
	}
	return out
}
