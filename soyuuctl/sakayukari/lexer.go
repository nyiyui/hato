package sakayukari

import (
	"bufio"
	"bytes"
	"errors"
	"unicode"
)

type TokenKind uint8

const (
	TKInvalid TokenKind = iota
	TKSkip
	TKIdent
	TKOp
	TKOpOC
	TKString
)

type Token struct {
	Kind  TokenKind
	Value []rune
}

type Lexer struct {
	r bufio.Reader
}

func (l *Lexer) Lex() (t Token, err error) {
	var one, two, three rune
	one, _, err = l.r.ReadRune()
	if err != nil {
		return
	}
	switch one {
	case ':':
		two, _, err = l.r.ReadRune()
		if err != nil {
			return
		}
		if two == '=' {
			t.Kind = TKOp
			t.Value = []rune{one, two}
			three, _, err = l.r.ReadRune()
			if err != nil {
				return
			}
			if !unicode.IsSpace(three) {
				t.Kind = TKInvalid
				err = errors.New("unexpected non-space after :=")
				return
			}
		} else if unicode.IsSpace(two) {
			t.Kind = TKOp
			t.Value = []rune{one}
		} else {
			err = errors.New("unexpected token")
			t.Kind = TKInvalid
			t.Value = []rune{one, two}
		}
	default:
		if unicode.IsSpace(one) {
			t.Kind = TKSkip
		} else if unicode.IsLetter(one) {
			// TODO: support proper (unicode and not just ' ') spaces here
			t.Kind = TKIdent
			var line []byte
			line, err = l.r.ReadBytes(' ')
			if err != nil {
				return
			}
			t.Value = bytes.Runes(line)
		} else {
			err = errors.New("unexpected token")
			t.Kind = TKInvalid
			t.Value = []rune{one}
		}
	}
	return
}
