package tokenizer

import "strings"

type Analyzer struct {
	tokenizer    *Tokenizer
	stem         bool
	normalizeNum bool
	dedupe       bool
}

func NewAnalyzer() *Analyzer {
	return &Analyzer{
		tokenizer:    New(),
		stem:         false,
		normalizeNum: false,
		dedupe:       true,
	}
}

func (a *Analyzer) WithStem() *Analyzer {
	a.stem = true
	return a
}

func (a *Analyzer) WithNumberNormalize() *Analyzer {
	a.normalizeNum = true
	return a
}

func (a *Analyzer) Analyze(text string) []string {
	if a == nil || a.tokenizer == nil {
		return nil
	}

	tokens := a.tokenizer.Tokenize(text)
	seen := make(map[string]struct{}, len(tokens))
	out := make([]string, 0, len(tokens))

	for _, tok := range tokens {
		if a.stem {
			tok = Stem(tok)
		}
		if a.normalizeNum {
			tok = NormalizeNumber(tok)
		}
		if a.dedupe {
			if _, ok := seen[tok]; ok {
				continue
			}
			seen[tok] = struct{}{}
		}
		out = append(out, tok)
	}
	return out
}

func (a *Analyzer) AnalyzeLower(text string) []string {
	tokens := a.Analyze(text)
	for i := range tokens {
		tokens[i] = strings.ToLower(tokens[i])
	}
	return tokens
}
