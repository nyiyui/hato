package yuuni

import (
	"fmt"

	"nyiyui.ca/soyuu/soyuuctl/sakayukari"
)

type MultiplexValue struct {
	Values []sakayukari.Value
}

func (v MultiplexValue) String() string {
	return fmt.Sprintf("muxv %#v", v.Values)
}

// TODO: make multiplex native-r
func Multiplex(src string, sinks []string) []sakayukari.Actor2 {
	actors := make([]sakayukari.Actor2, 0, len(sinks))
	for i, sink := range sinks {
		actors = append(actors, sakayukari.Actor2{
			DependsOn: []string{src},
			UpdateFunc: func(self *sakayukari.Actor, gsm sakayukari.GraphStateMap, gs sakayukari.GraphState) (updated sakayukari.Value) {
				v := gs.States[gsm[src]].(MultiplexValue)
				return v.Values[i]
			},
			Comment: fmt.Sprintf("multiplex %#v: %d %s", sinks, i, sink),
		})
	}
	return actors
}
