package conn

import (
	"bufio"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	. "nyiyui.ca/hato/sakayukari"
)

type handlerBreakbeam struct{}

func (_ handlerBreakbeam) HandleConn(a Actor, c *Conn) {
	reader := bufio.NewReader(c.F)
	for {
		lineRaw, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("%s: read line: %s", c.Path, err)
			continue
		}
		if !strings.HasPrefix(lineRaw, " D") {
			continue
		}
		line := lineRaw[2:]
		values, monotonic, err := parse(line)
		if err != nil {
			log.Printf("handlerBreakbeam: %s", err)
			continue
		}
		v := ValSeen{Monotonic: monotonic, Sensors: make([]ValSeenSensor, 0, len(values))}
		for sensor, value := range values {
			v.Sensors = append(v.Sensors, ValSeenSensor{
				Name: sensor,
				Seen: value,
			})
			log.Printf("sense %s %t", sensor, value)
		}
		a.OutputCh <- Diffuse1{Value: v}
	}
}

func (_ handlerBreakbeam) NewBlankActor() Actor {
	return Actor{
		Comment:  "blank handlerBreakbeam",
		OutputCh: make(chan Diffuse1),
		Type: ActorType{
			Output: true,
		},
	}
}

func parse(line string) (values map[string]bool, monotonic int64, err error) {
	values = map[string]bool{}
	for i := 0; i < len(line); i++ {
		switch {
		case line[i] == 'T':
			j := strings.IndexFunc(line[i+1:], func(r rune) bool { return r >= 'A' && r <= 'Z' })
			if j == -1 {
				j = len(line) - i
			}
			monotonic, err = strconv.ParseInt(strings.TrimSpace(line[i+1:i+j]), 10, 64)
			if err != nil {
				err = fmt.Errorf("parse: T: %s", err)
				return
			}
			i += j
		default:
			values[string(line[i])] = line[i+1] == '1'
			i++
		}
	}
	return
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
// positive velocity is A → B
func Velocity2(breakbeam ActorRef, sensorA, sensorB string, interval Length, position int64) Actor {
	actor := Actor{
		Comment:  fmt.Sprintf("velocity2 %s", breakbeam),
		InputCh:  make(chan Diffuse1),
		OutputCh: make(chan Diffuse1),
		Inputs:   []ActorRef{breakbeam},
		Type: ActorType{
			Input:       true,
			LinearInput: true,
			Output:      true,
		},
	}
	go func() {
		s := newVelocity2State()
	ActorLoop:
		for d := range actor.InputCh {
			if d.Origin != breakbeam {
				panic(fmt.Sprintf("unknown origin %s", d.Origin))
			}
			now := time.Now()
			v := d.Value.(ValSeen)
			single := velocity2Single{
				Monotonic: v.Monotonic,
			}
			for _, sensor := range v.Sensors {
				switch sensor.Name {
				case sensorA:
					single.PointA = sensor.Seen
				case sensorB:
					single.PointB = sensor.Seen
				}
			}
			if s.History[0].PointA == single.PointA && s.History[0].PointB == single.PointB {
				// equivalent to before
				continue ActorLoop
			}
			s.Shift(single)
			log.Printf("s %v", s.History)
			h0 := single
			h1 := s.History[1]
			h2 := s.History[2]

			a := ValAttitude{
				Monotonic: h0.Monotonic,
				Time:      now,
			}

			// === Longer-than-interval cars

			//   --> true direction/velocity means A → B
			//   A B
			// 1 x o (1 change before)
			// 0 x x (now)
			if h1.PointA != h1.PointB && h0.PointA == false && h0.PointB == false {
				a.Front = true
				dt := h0.Monotonic - h1.Monotonic
				if dt != 0 {
					if h1.PointA {
						// train is near pointA now
						// A---B
						//     <
						// <===[
						a.Position = position
						a.Velocity = -a.Velocity
					} else {
						// train is near pointB now
						// A---B
						// >
						// ]===>
						a.Position = position + interval
					}
					a.Velocity = interval * 1000 / dt
					log.Printf("att1l %s", a)
					actor.OutputCh <- Diffuse1{Value: a}
				}
			}
			//   A B
			// 2 x o
			// 1 x x
			// 0 o x
			if h2.PointA != h2.PointB && h1.PointA == false && h1.PointB == false && h0.PointA != h0.PointB && h0.PointA != h2.PointA {
				a.Front = false
				dt := h1.Monotonic - h2.Monotonic
				if dt != 0 {
					carsLength := interval*(h0.Monotonic-h1.Monotonic)/dt + interval
					if h0.PointA {
						// train is near pointB now
						//   A---B
						//       <
						//   <====[
						// <====[
						a.Position = position + interval - carsLength
						a.Velocity = -a.Velocity
					} else {
						// train is near pointA now
						// A---B
						// >
						// ]====>
						//  ]====>
						a.Position = position + carsLength
					}
					a.Velocity = interval * 1000 / dt
					log.Printf("att2l %s cars%d pos%d", a, carsLength, a.Position)
					actor.OutputCh <- Diffuse1{Value: a}
				}
			}

			/*
				// === Shorter-than-interval Cars

				//   A B
				// 2 o x
				// 1 o o
				// 0 x o
				if h2.PointA != h2.PointB && h1.PointA == true && h1.PointB == true && h0.PointA != h0.PointB {
					a.Front = true
					dt := h2.Monotonic - h0.Monotonic
					if dt != 0 {
						a.Velocity = interval * 1000 / dt
						if h0.PointA {
							// A---B
							//     <=[
							//    <=[  // ignored
							//   <=[   // ignored
							//  <=[
							// <=[
							a.Position = position
							a.Velocity = -a.Velocity
						} else {
							//   A---B
							// ]=>
							//  ]=>    // ignored
							//   ]=>   // ignored
							//    ]=>
							//     ]=>
							a.Position = position + interval
						}
						log.Printf("att1s %s", a)
						actor.OutputCh <- Diffuse1{Value: a}
					}
				}
				// TODO: Shorter-than-interval carsLength
			*/
		}
	}()
	return actor
}
