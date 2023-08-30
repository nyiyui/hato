package treewalk

import (
	"fmt"

	"nyiyui.ca/hato/sakayukari2/parser"
)

var builtins = map[string]Object{
	"print": &GoFunc{func(val interface{}) {
		fmt.Printf("%s", val)
	}},
	"exec": exec{},
}

type Instance struct {
	parent *Instance
	names  map[string]Object
}

func (i *Instance) Exec(n parser.Node) (parser.Node, error) {
	switch n := n.(type) {
	case List:
		if len(n.Content) == 0 {
			return Unit, nil
		}
		callee := n.Content[0]
		f, ok := callee.(Callable)
		if !ok {
			return nil, fmt.Errorf("%s not callable", callee)
		}
		return f.Call(i, n.Content[1:])
	case String, Int:
		return n, nil
	case Atom:
		n2, ok := i.lookup(n.Content)
		if !ok {
			return nil, fmt.Errorf("%s not defined", n.Content)
		}
		return n2, nil
	}
}

func (i *Instance) lookup(name string) (n parser.Node, ok bool) {
	n, ok = builtins[name]
	if ok {
		return
	}
	n, ok = i.names[name]
	if ok {
		return
	}
	if i.parent != nil {
		return i.parent.lookup(name)
	}
	return nil, false
}
