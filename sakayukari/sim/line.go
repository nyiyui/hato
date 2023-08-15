package sim

import (
	"fmt"

	. "nyiyui.ca/hato/sakayukari"
	"nyiyui.ca/hato/sakayukari/conn"
)

func (s *Simulation) newLine(c conn.Id) *Actor {
	a := &Actor{
		Comment:  fmt.Sprintf("sim-line-%s", c),
		InputCh:  make(chan Diffuse1),
		OutputCh: make(chan Diffuse1),
		Type:     ActorType{Input: true, LinearInput: true, Output: true},
	}
	// for ValCurrent
	go func() {
	}()
	// for ReqLine, ReqSwitch
	go func() {
		for d := range a.InputCh {
			switch val := d.Value.(type) {
			case conn.ReqLine:
			case conn.ReqSwitch:
			}
		}
	}()
	return a
}
