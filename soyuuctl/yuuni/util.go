package yuuni

import (
	"fmt"
	"log"

	"nyiyui.ca/soyuu/soyuuctl/conn"
	"nyiyui.ca/soyuu/soyuuctl/sakayukari"
)

type hysteresisState struct {
	Prev conn.ValSeen
}

func hysteresis(holdoff int64) sakayukari.Actor2 {
	state := new(hysteresisState)
	actor := sakayukari.Actor2{
		UpdateFunc: func(self *sakayukari.Actor, gsm sakayukari.GraphStateMap, gs sakayukari.GraphState) (updated sakayukari.Value) {
			cur := gs.States[self.DependsOn[0]].(conn.ValSeen)
			log.Printf("hys %d > %d", cur.Monotonic-state.Prev.Monotonic, holdoff)
			if cur.Seen != state.Prev.Seen && cur.Monotonic-state.Prev.Monotonic > holdoff {
				state.Prev = cur
			}
			return state.Prev
		},
	}
	return actor
}

type velocityState struct {
	Point string
	Seen  conn.ValSeen
	// Seen   []conn.ValSeen
	Result *conn.ValAttitude
}

func (s *velocityState) WaitingFor(point string, seen conn.ValSeen) {
	s.Point = point
	s.Seen = seen
}

func (s *velocityState) Reset() {
	s.Point = ""
	s.Seen = conn.ValSeen{}
	s.Result = nil
}

func velocity(pointA, pointB string) sakayukari.Actor2 {
	state := new(velocityState)
	// state.Seen = make([]conn.ValSeen, 2)
	return sakayukari.Actor2{
		DependsOn: []string{pointA, pointB},
		UpdateFunc: func(self *sakayukari.Actor, gsm sakayukari.GraphStateMap, gs sakayukari.GraphState) (updated sakayukari.Value) {
			valueA := gs.States[gsm[pointA]]
			valueB := gs.States[gsm[pointB]]
			if valueA == nil || valueB == nil {
				log.Print("nil")
				return nil
			}
			points := map[string]conn.ValSeen{
				"A": valueA.(conn.ValSeen),
				"B": valueB.(conn.ValSeen),
			}
			// vals := []conn.ValSeen{
			// 	valueA.(conn.ValSeen),
			// 	valueB.(conn.ValSeen),
			// }
			for point, seen := range points {
				log.Printf("seenRaw %s %#v", point, seen)
				if !seen.Seen && state.Point == point {
					log.Print("reset")
					state.Reset()
				} else if seen.Seen && state.Point != point {
					log.Printf("seen %s", point)
					dt := seen.Monotonic - state.Seen.Monotonic
					direction := point == "B" // true if A → B
					if !direction {
						dt = -dt
					}
					if dt == 0 {
						log.Print("divby0")
					} else {
						state.Result = &conn.ValAttitude{
							Velocity: 248 * 1000 * 1000 / dt,
						}
					}
				} else if seen.Seen && (state.Point == "" || state.Point == point) {
					log.Printf("waiting; seen %s", seen)
					state.WaitingFor(point, seen)
				}
			}
			log.Printf("state %#v", state)
			return state.Result
		},
		Comment: fmt.Sprintf("velocity %s %s", pointA, pointB),
	}
}
