package rikien

import "io"

type Lexer struct {
	r io.Reader
}

func NewLexer(r io.Reader) *Lexer { return &Lexer{r: r} }
func (l *Lexer) Lex() Token {
	return Token{}
}
