package treewalk

import (
	"errors"
	"fmt"
	"reflect"
)

type Object interface {
	fmt.Stringer
}

type unit struct{}

var Unit = unit{}

func (u unit) String() string { return "()" }

type Callable interface {
	Call(i *Instance, args []Object) (Object, error)
}

type GoFunc struct {
	Func interface{}
}

func (g *GoFunc) String() string {
	return fmt.Sprintf("[GoFunc %#v", g.Func)
}

func (g *GoFunc) Call(i *Instance, args []Object) (Object, error) {
	f := reflect.ValueOf(g.Func)
	ft := f.Type()
	if ft.NumIn() != len(args) {
		return nil, fmt.Errorf("expected %d args but got %d", ft.NumIn(), len(args))
	}
	switch ft.NumOut() {
	case 0, 1:
	case 2:
		if ft.Out(1) != reflect.TypeOf(errors.New("")) {
			return nil, fmt.Errorf("internal: 2nd return value must be error, not %s", ft.Out(1))
		}
	default:
		return nil, fmt.Errorf("internal: number of return values must be 0, 1, or 2")
	}
	args2 := make([]reflect.Value, ft.NumIn())
	for i := 0; i < ft.NumIn(); i++ {
		at := ft.In(i)
		if at == reflect.TypeOf(interface{}(nil)) {
			args2[i] = reflect.ValueOf(args[i])
		} else {
			return nil, fmt.Errorf("internal: non-any argument types not supported (yet)")
		}
	}
	ret := f.Call(args2)
	switch len(ret) {
	case 0:
		return nil, nil
	case 1:
		val := ret[0].Interface()
		return InterfaceToObject(val), nil
	case 2:
		val := ret[0].Interface()
		err := ret[1].Interface().(error)
		if err != nil {
			return nil, err
		}
		return InterfaceToObject(val), nil
	default:
		panic("unreachable")
	}
}

type GoInterface struct {
	val interface{}
}

func (g *GoInterface) String() string {
	return fmt.Sprintf("[GoInterface %#v", g)
}

func InterfaceToObject(val interface{}) Object {
	switch val := val.(type) {
	default:
		return &GoInterface{val: val}
	}
}

type exec struct {
}

func (e exec) String() string {
	return "exec"
}

func(e exec)	Call(i *Instance, args []Object) (Object, error){
	i.Exec(
}