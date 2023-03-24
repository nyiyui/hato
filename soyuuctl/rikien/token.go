package rikien

type Token struct {
	Kind    TokenKind
	Content []byte
}

type TokenKind uint

const (
	TKInvalid TokenKind = iota
	TKOp
	TKIdent
	TKString
)
