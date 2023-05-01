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

type velocity2Single struct {
	Monotonic int64
	PointA    bool
	PointB    bool
}

func (s *velocity2Single) String() string {
	res := make([]byte, 2)
	if s.PointA {
		res[0] = '1'
	} else {
		res[0] = '0'
	}
	if s.PointB {
		res[1] = '1'
	} else {
		res[1] = '0'
	}
	return fmt.Sprintf("%s %d", res, s.Monotonic)
}

type velocity2State struct {
	History []velocity2Single
}

func newVelocity2State() *velocity2State {
	return &velocity2State{
		History: make([]velocity2Single, 3),
	}
}

func (s *velocity2State) Shift(single velocity2Single) {
	for i := len(s.History) - 1; i > 0; i-- {
		s.History[i] = s.History[i-1]
	}
	s.History[0] = single
}

// position is where pointA is
func velocity2(pointA, pointB string, interval conn.Length, position int64) sakayukari.Actor2 {
	s := newVelocity2State()
	return sakayukari.Actor2{
		DependsOn: []string{pointA, pointB},
		UpdateFunc: func(self *sakayukari.Actor, gsm sakayukari.GraphStateMap, gs sakayukari.GraphState) (updated sakayukari.Value) {
			valueA := gs.States[gsm[pointA]]
			valueB := gs.States[gsm[pointB]]
			if valueA == nil || valueB == nil {
				log.Print("nil")
				return nil
			}
			{
				valueA := valueA.(conn.ValSeen)
				valueB := valueB.(conn.ValSeen)
				single := velocity2Single{
					Monotonic: valueA.Monotonic,
					PointA:    valueA.Seen,
					PointB:    valueB.Seen,
				}
				if s.History[0].PointA == single.PointA && s.History[0].PointB == single.PointB {
					// equivalent to before
				} else {
					s.Shift(single)
					log.Printf("s %v", s.History)
					h0 := single
					h1 := s.History[1]
					h2 := s.History[2]
					//   --> true direction means A → B
					//   A B
					// 1 x o (1 change before)
					// 0 x x (now)
					if h1.PointA != h1.PointB && h0.PointA == false && h0.PointB == false {
						a := conn.ValAttitude{
							Monotonic: h0.Monotonic,
							Front:     true,
						}
						if h1.PointA {
							// train is at pointA now
							// A---B
							//     <
							// <===[
							a.Position = position
						} else {
							// train is at pointB now
							// A---B
							// >
							// ]===>
							a.Position = position + interval
						}
						dt := h0.Monotonic - h1.Monotonic
						if dt != 0 {
							a.Velocity = interval * 1000 / dt
							if h1.PointA {
								a.Velocity = -a.Velocity
							}
							log.Printf("att1 %s", a)
							return a
						}
					}
					//   A B
					// 2 x o
					// 1 x x
					// 0 o x
					if h2.PointA != h2.PointB && h1.PointA == false && h1.PointB == false && h0.PointA != h0.PointB && h0.PointA != h2.PointA {
						a := conn.ValAttitude{
							Monotonic: h0.Monotonic,
							Front:     false,
						}
						dt := h1.Monotonic - h2.Monotonic
						if dt != 0 {
							carsLength := interval*(h0.Monotonic-h1.Monotonic)/dt + interval
							if h0.PointA {
								// train is at pointB now
								// A---B
								//     <
								// <===[
								// ===[
								a.Position = position + interval
							} else {
								// train is at pointA now
								// A---B
								// >
								// ]===>
								//  ]===
								a.Position = position
							}
							a.Velocity = interval * 1000 / dt
							if !h0.PointA {
								a.Velocity = -a.Velocity
							}
							log.Printf("att2 %s cars%d pos%d", a, carsLength, a.Position)
						}
					}
				}
			}
			return nil
		},
		Comment: fmt.Sprintf("velocity %s %s", pointA, pointB),
	}
}
