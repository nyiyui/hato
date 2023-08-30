package parser

import (
	"bytes"
	"fmt"
	"strconv"
)

type Node interface {
	fmt.Stringer
}

type List struct {
	Content []Node
}

func (l List) String() string {
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "[%d", len(l.Content))
	for _, node := range l.Content {
		fmt.Fprintf(buf, " %s", node)
	}
	fmt.Fprint(buf, "]")
	return buf.String()
}

type String struct {
	Content string
}

func (s String) String() string {
	return strconv.Quote(s.Content)
}

type Atom struct {
	Content string
}

func (a Atom) String() string {
	return fmt.Sprintf("%s", a.Content)
}

type Int struct {
	Content int64
}

func (i Int) String() string { return strconv.FormatInt(i.Content, 10) }
