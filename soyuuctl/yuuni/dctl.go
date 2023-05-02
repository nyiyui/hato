package yuuni

import (
	"fmt"
	"log"

	"nyiyui.ca/soyuu/soyuuctl/conn"
	"nyiyui.ca/soyuu/soyuuctl/sakayukari"
)

type directControlState struct {
	Keys     []string
	Values   []int
	Selected int
}

func directControl(keyboard string, mapping map[string]string) sakayukari.Actor2 {
	state := new(directControlState)
	state.Keys = make([]string, 0, len(mapping))
	for key := range mapping {
		state.Keys = append(state.Keys, key)
	}
	state.Values = make([]int, len(mapping))
	return sakayukari.Actor2{
		DependsOn: append(state.Keys, keyboard),
		UpdateFunc: func(self *sakayukari.Actor, gsm sakayukari.GraphStateMap, gs sakayukari.GraphState) (updated sakayukari.Value) {
			ev := gs.States[gsm[keyboard]].(EventValue).Value
			changed := false
			switch ev.ID {
			case "<Right>":
				state.Selected++
				state.Selected %= len(state.Keys)
			case "<Left>":
				state.Selected--
				state.Selected %= len(state.Keys)
			case "<Up>":
				state.Values[state.Selected] += 3
				changed = true
			case "<Down>":
				state.Values[state.Selected] -= 3
				changed = true
			}
			if state.Values[state.Selected] > 255 {
				state.Values[state.Selected] = 255
			} else if state.Values[state.Selected] < -255 {
				state.Values[state.Selected] = -255
			}
			log.Printf("STATE %t %#v", changed, state)
			if changed {
				val := state.Values[state.Selected]
				p := uint8(val)
				if val < 0 {
					p = uint8(-val)
				}
				return conn.ReqLine{
					Brake:     val == 0,
					Direction: val > 0,
					Power:     p,
				}
			}
			return nil
		},
		SideEffects: true,
		Comment:     fmt.Sprintf("dctl from %s to %#v", keyboard, mapping),
	}
}

type directControl2State struct {
	Keys     []string
	Values   []int
	Selected int
}

func directControl2(id string, keyboard string, mapping map[string]string) map[string]sakayukari.Actor2 {
	state := new(directControl2State)
	state.Keys = make([]string, 0, len(mapping))
	for key := range mapping {
		state.Keys = append(state.Keys, key)
	}
	state.Values = make([]int, len(mapping))
	main := sakayukari.Actor2{
		DependsOn: append(state.Keys, keyboard),
		UpdateFunc: func(self *sakayukari.Actor, gsm sakayukari.GraphStateMap, gs sakayukari.GraphState) (updated sakayukari.Value) {
			ev := gs.States[gsm[keyboard]].(EventValue).Value
			changed := false
			switch ev.ID {
			case "<Right>":
				state.Selected++
				state.Selected %= len(state.Keys)
			case "<Left>":
				state.Selected--
				state.Selected %= len(state.Keys)
			case "<Up>":
				state.Values[state.Selected] += 3
				changed = true
			case "<Down>":
				state.Values[state.Selected] -= 3
				changed = true
			}
			if state.Values[state.Selected] > 255 {
				state.Values[state.Selected] = 255
			} else if state.Values[state.Selected] < -255 {
				state.Values[state.Selected] = -255
			}
			log.Printf("STATE %t %#v", changed, state)
			if changed {
				val := state.Values[state.Selected]
				p := uint8(val)
				if val < 0 {
					p = uint8(-val)
				}
				v := MultiplexValue{
					Values: make([]sakayukari.Value, len(state.Keys)),
				}
				v.Values[state.Selected] = conn.ReqLine{
					Brake:     val == 0,
					Direction: val > 0,
					Power:     p,
				}
				return v
			}
			return nil
		},
		SideEffects: true,
		Comment:     fmt.Sprintf("dctl from %s to %#v", keyboard, mapping),
	}
	actors := Multiplex(id, state.Keys)
	res := map[string]sakayukari.Actor2{}
	for i, actor := range actors {
		res[fmt.Sprintf("%s-%d", id, i)] = actor
	}
	res[id] = main
	return res
}
