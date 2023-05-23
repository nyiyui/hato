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
		i, sink := i, sink
		actors = append(actors, sakayukari.Actor2{
			DependsOn: []string{src},
			UpdateFunc: func(self *sakayukari.Actor, gsm sakayukari.GraphStateMap, gs sakayukari.GraphState) (updated sakayukari.Value) {
				vRaw := gs.States[gsm[src]]
				if vRaw == nil {
					// log.Printf("MULTIPLEX %s nil", src)
					return sakayukari.SpecialIgnore
				}
				v := vRaw.(MultiplexValue)
				// log.Printf("MULTIPLEX %s i%d value %#v", src, i, v)
				return v.Values[i]
			},
			Comment: fmt.Sprintf("multiplex %#v: %d %s", sinks, i, sink),
		})
	}
	return actors
}
