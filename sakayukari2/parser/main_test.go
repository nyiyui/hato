package parser

import (
	"bytes"
	"fmt"
	"testing"
)

// TestParser tests whether the parser doesn't error out on certain test cases.
// Note that it doesn't test whether the output is correct.
func TestParser(t *testing.T) {
	type setup struct {
		src string
	}
	setups := []setup{
		{"123"},
		{"0b111"},
		{"0o777"},
		{"0777"},
		{"atom"},
		{"()"},
		{"(atom)"},
		{"(a b)"},
		{`"string"`},
		{`""`},
		{`"\""`},
	}
	for i, s := range setups {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			p := New(bytes.NewBufferString(s.src))
			n, err := p.Parse()
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("node: %s", n)
		})
	}
}
