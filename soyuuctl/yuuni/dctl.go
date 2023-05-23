package yuuni

import (
	"fmt"
	"log"
	"sort"
	"time"

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
				state.Selected += len(state.Keys)
				state.Selected %= len(state.Keys)
			case "<Left>":
				state.Selected--
				state.Selected += len(state.Keys)
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
	Keys       []string
	Values     []int
	ResetAfter []*time.Time
	Selected   int
}

type MappingType int

const (
	MappingInvalid MappingType = iota
	MappingLine
	MappingSwitch
)

type MappingValue struct {
	Name string
	Type MappingType
}

type InputValue struct {
	Line  string
	Value int
}

func (i InputValue) String() string {
	return fmt.Sprintf("InputValue %s %d", i.Line, i.Value)
}

func directControl2(id string, keyboard string, mapping map[string]MappingValue, actors map[string]sakayukari.Actor2, input string) {
	state := new(directControl2State)
	state.Keys = make([]string, 0, len(mapping))
	nameToIndex := map[string]int{}
	for key := range mapping {
		nameToIndex[key] = len(state.Keys)
		state.Keys = append(state.Keys, key)
	}
	sort.Strings(state.Keys)
	state.Values = make([]int, len(mapping))
	state.ResetAfter = make([]*time.Time, len(mapping))
	main := sakayukari.Actor2{
		DependsOn: append(state.Keys, keyboard), //, input),
		UpdateFunc: func(self *sakayukari.Actor, gsm sakayukari.GraphStateMap, gs sakayukari.GraphState) (updated sakayukari.Value) {
			changed := false
			if input != "" {
				inpRaw := gs.States[gsm[input]]
				if inpRaw != nil && inpRaw != sakayukari.SpecialIgnore {
					inp := inpRaw.(InputValue)
					// log.Printf("directControl2: set input %#v", inp)
					state.Values[nameToIndex[inp.Line]] = inp.Value
					changed = true
				}
			}
			evRaw := gs.States[gsm[keyboard]]
			if evRaw != nil {
				ev := evRaw.(EventValue).Value

				// handle input
				switch ev.ID {
				case "<Right>":
					state.Selected++
					state.Selected %= len(state.Keys)
				case "<Left>":
					state.Selected--
					state.Selected += len(state.Keys)
					state.Selected %= len(state.Keys)
				case "<Up>":
					switch mapping[state.Keys[state.Selected]].Type {
					case MappingLine:
						state.Values[state.Selected] += 3
					case MappingSwitch:
						state.Values[state.Selected] = 255
						a := time.Now().Add(1 * time.Second)
						state.ResetAfter[state.Selected] = &a
					}
					changed = true
				case "<Down>":
					switch mapping[state.Keys[state.Selected]].Type {
					case MappingLine:
						state.Values[state.Selected] -= 3
					case MappingSwitch:
						state.Values[state.Selected] = -255
						a := time.Now().Add(1 * time.Second)
						state.ResetAfter[state.Selected] = &a
					}
					changed = true
				}
			}
			// handle ResetAfter
			for i := range state.ResetAfter {
				if state.ResetAfter[i] == nil {
					continue
				}
				if time.Now().After(*state.ResetAfter[i]) {
					state.ResetAfter[i] = nil
					state.Values[i] = 0
				}
			}
			// clamp
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
				for i := range v.Values {
					v.Values[i] = sakayukari.SpecialIgnore
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
	muxActors := Multiplex(id, state.Keys)
	for i, muxActor := range muxActors {
		sink := mapping[state.Keys[i]].Name
		name := fmt.Sprintf("%s-%d", id, i)
		actors[name] = muxActor
		recv, ok := actors[sink]
		if !ok {
			panic(fmt.Sprintf("actors[%s] for sink not found", sink))
		}
		recv.DependsOn = append(recv.DependsOn, name)
		if recv.Comment == "" {
			recv.Comment = fmt.Sprintf("REPLACEMENT %s %d %#v", id, i, recv)
		}
		actors[sink] = recv
	}
	actors[id] = main
}
