package parser

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
)

type parser struct {
	// lexer fields
	r             *bufio.Reader
	intBuf        []byte
	strBuf        []byte
	strEscapeNext bool
	atomBuf       []byte

	// lexer-parser fields
	parseCh chan Token
}

func New(src io.Reader) *parser {
	return &parser{
		r:       bufio.NewReader(src),
		parseCh: make(chan Token),
	}
}

type Token struct {
	Type       TokenType
	Int        int64
	StringAtom string
	Error      error
}

type TokenType int

const (
	TTInvalid TokenType = iota
	TTError
	TTOpen
	TTClose
	TTString
	TTAtom
	TTInt
)

func (p *parser) Parse() (Node, error) {
	go func() {
		for {
			err := p.lex()
			if err != nil {
				p.parseCh <- Token{Type: TTError, Error: err}
			}
		}
	}()
	return p.handleCh()
}

func (p *parser) lex() (err error) {
	b, err := p.r.ReadByte()
	if err == io.EOF {
		err = p.lexSplit(0)
		if err != nil {
			return
		}
		return io.EOF
	} else if err != nil {
		return
	}
	switch {
	case b == '(':
		p.lexSplit(0)
		err = p.parse(Token{
			Type: TTOpen,
		})
		if err != nil {
			return
		}
	case b == ')':
		p.lexSplit(0)
		err = p.parse(Token{
			Type: TTClose,
		})
		if err != nil {
			return
		}
	case '0' <= b && b <= '9':
		if p.intBuf == nil {
			p.intBuf = make([]byte, 0, 1)
		}
		p.intBuf = append(p.intBuf, b)
	case b == ' ' || b == '\t' || b == '\n':
		err = p.lexSplit(b)
		if err != nil {
			return
		}
	case b == '"':
		if len(p.strBuf) == 0 {
			p.strBuf = make([]byte, 1)
			p.strBuf[0] = b
			p.strEscapeNext = false
		} else if p.strEscapeNext {
			p.strBuf = append(p.strBuf, b)
			p.strEscapeNext = false
		} else {
			var s string
			p.strBuf = append(p.strBuf, '"')
			s, err = strconv.Unquote(string(p.strBuf))
			if err != nil {
				err = fmt.Errorf("parse str: %w", err)
				return
			}
			p.strBuf = nil
			err = p.parse(Token{
				Type:       TTString,
				StringAtom: s,
			})
			if err != nil {
				return
			}
		}
	default:
		switch {
		case len(p.intBuf) == 1:
			p.intBuf = append(p.intBuf, b)
		case len(p.strBuf) != 0:
			if p.strEscapeNext {
				p.strEscapeNext = false
			}
			p.strBuf = append(p.strBuf, b)
			if b == '\\' {
				p.strEscapeNext = true
			}
		case b == 0:
		default:
			if p.atomBuf == nil {
				p.atomBuf = make([]byte, 0, 1)
			}
			p.atomBuf = append(p.atomBuf, b)
		}
	}
	return
}

func (p *parser) lexSplit(b byte) (err error) {
	switch {
	case len(p.intBuf) != 0:
		var i int64
		i, err = strconv.ParseInt(string(p.intBuf), 0, 64)
		if err != nil {
			err = fmt.Errorf("parse int: %w", err)
			return
		}
		p.intBuf = nil
		err = p.parse(Token{
			Type: TTInt,
			Int:  i,
		})
		if err != nil {
			return
		}
	case len(p.strBuf) != 0:
		if b == '\n' {
			return errors.New("string cannot continue after a newline")
		}
		p.strBuf = append(p.strBuf, b)
	case len(p.atomBuf) != 0:
		atomBuf := p.atomBuf
		p.atomBuf = nil
		err = p.parse(Token{
			Type:       TTAtom,
			StringAtom: string(atomBuf),
		})
		if err != nil {
			return
		}
	}
	return
}

func (p *parser) parse(tok Token) (err error) {
	p.parseCh <- tok
	return nil
}

var ttCloseError = errors.New("internal: TTClose")

func (p *parser) handleCh() (Node, error) {
	for tok := range p.parseCh {
		switch tok.Type {
		case TTError:
			err := tok.Error
			if err == io.EOF {
				return nil, errors.New("unexpected EOF")
			}
			if err != nil {
				return nil, err
			}
			return nil, errors.New("TTError with nil error")
		case TTOpen:
			n := List{Content: make([]Node, 0, 1)}
			for i := 0; true; i++ {
				n2, err := p.handleCh()
				if err == ttCloseError {
					break
				} else if err != nil {
					return nil, fmt.Errorf("list index %d: %w", i, err)
				}
				n.Content = append(n.Content, n2)
			}
			return n, nil
		case TTClose:
			return nil, ttCloseError
		case TTString:
			return String{
				Content: tok.StringAtom,
			}, nil
		case TTAtom:
			return Atom{
				Content: tok.StringAtom,
			}, nil
		case TTInt:
			return Int{
				Content: tok.Int,
			}, nil
		default:
			return nil, fmt.Errorf("invalid token %#v", tok)
		}
	}
	panic("unreachable")
}
