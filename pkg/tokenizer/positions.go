package tokenizer

type Token struct {
	Text string

	Position int
}

func (t *Tokenizer) TokenizeWithPosition(text string) []Token {
	tokens := t.Tokenize(text)
	out := make([]Token, 0, len(tokens))
	for i, tok := range tokens {
		out = append(out, Token{Text: tok, Position: i})
	}
	return out
}

func TokenStream(tokens []string) []Token {
	out := make([]Token, 0, len(tokens))
	for i, tok := range tokens {
		out = append(out, Token{Text: tok, Position: i})
	}
	return out
}
