package conn

import (
	"bufio"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	. "nyiyui.ca/hato/sakayukari"
)

type sensor struct {
	ID rune
	// Position of the sensor in µm from an arbitrary point of the sensor
	Position int64
}

type handlerBreakbeam struct{}

func (_ handlerBreakbeam) HandleConn(a Actor, c *Conn) {
	reader := bufio.NewReader(c.F)
	_, err := fmt.Fprint(c.F, "J\n")
	if err != nil {
		log.Printf("%s: J: write line: %s", c.Path, err)
		return
	}
	var sensors []sensor
	{
		jRaw, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("%s: J: read line: %s", c.Path, err)
			return
		}
		log.Printf("breakbeam: J: %s", jRaw)
		parts := strings.SplitN(jRaw, ";", 2)
		sensorRaws := strings.Split(parts[1], " ")
		sensors = make([]sensor, 0, len(sensorRaws))
		for i, sensorRaw := range sensorRaws {
			// NOTE: assume JAP248000 form (A = sensor id, 248000 = position in µm)
			pos, err := strconv.ParseInt(strings.TrimSpace(sensorRaw[3:]), 10, 64)
			if err != nil {
				log.Printf("%s: J: parse %d: %s", c.Path, i, err)
				return
			}
			sensors = append(sensors, sensor{
				ID:       rune(sensorRaw[1]),
				Position: pos,
			})
		}
	}

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
			var pos int64 = -1
			for _, s := range sensors {
				if s.ID == rune(sensor[0]) {
					pos = s.Position
				}
			}
			v.Sensors = append(v.Sensors, ValSeenSensor{
				Name:     sensor,
				Seen:     value,
				Position: pos,
			})
		}
		a.OutputCh <- Diffuse1{Value: v}
		// TODO: velocity2 hangs
		/*
		   23/06/01 08:18:53 connecting to /dev/ttyACM1
		   /dev/ttyACM0: ready
		   /dev/ttyACM0: send 1 I
		   /dev/ttyACM0: recv 21 b' Isoyuu-line-mega-0\r\n'
		   2023/06/01 08:18:53 handling /dev/ttyACM0 soyuu-line-mega-0/-
		   /dev/ttyACM1: ready
		   /dev/ttyACM1: send 1 I
		   /dev/ttyACM1: recv 32 b' Isoyuu-breakbeam/itsybitsy0/0\r\n'
		   2023/06/01 08:18:53 handling /dev/ttyACM1 soyuu-breakbeam/itsybitsy0-0
		   /dev/ttyACM1: send 1 J
		   /dev/ttyACM1: recv 57 b' Isoyuu-breakbeam/itsybitsy0/0;JAP0 JBP248000 JCP496000\r\n'
		   2023/06/01 08:18:53 breakbeam: J:  Isoyuu-breakbeam/itsybitsy0/0;JAP0 JBP248000 JCP496000
		   /dev/ttyACM1: recv 19 b' DA1B1C0T21572396\r\n'
		   2023/06/01 08:18:59 outputting
		   2023/06/01 08:18:59 output ok
		   2023/06/01 08:18:59 w0s [{21572396 true true} {0 false false} {0 false false}]
		   2023/06/01 08:18:59 w1s [{21572396 true false} {0 false false} {0 false false}]
		   /dev/ttyACM1: recv 19 b' DA1B0C0T21574178\r\n'
		   2023/06/01 08:19:00 outputting
		   2023/06/01 08:19:00 output ok
		   2023/06/01 08:19:00 w0s [{21574178 true false} {21572396 true true} {0 false false}]
		   2023/06/01 08:19:00 w1s [{21574178 false false} {21572396 true false} {0 false false}]
		   2023/06/01 08:19:00 ATT1l w1 attitude(0 248000µm -139.169mm/s -75.15126km/h 21574178 nf)
		   2023/06/01 08:19:00 vel -139169
		   2023/06/01 08:19:00 curPos 248000
		   /dev/ttyACM1: recv 19 b' DA0B0C0T21575394\r\n'
		   2023/06/01 08:19:02 outputting
		   2023/06/01 08:19:02 output ok
		   2023/06/01 08:19:02 w0s [{21575394 false false} {21574178 true false} {21572396 true true}]
		   2023/06/01 08:19:02 w1equiv {21574178 false false} and {21575394 false false}
		   2023/06/01 08:19:02 ATT1l w0 attitude(0 0µm -203.947mm/s -110.13138km/h 21575394 nf)
		   2023/06/01 08:19:02 ATT1l w1 attitude(0 248000µm -139.169mm/s -75.15126km/h 21574178 nf)
		   2023/06/01 08:19:02 vel -203947
		   2023/06/01 08:19:02 curPos 0
		   /dev/ttyACM1: recv 19 b' DA0B0C1T21576967\r\n'
		   2023/06/01 08:19:03 outputting
		   /dev/ttyACM1: recv 19 b' DA0B1C1T21578206\r\n'
		   /dev/ttyACM1: recv 19 b' DA1B1C1T21580509\r\n'
		*/
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
		case line[i] >= 'A' && line[i] <= 'Z':
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
	return fmt.Sprintf("%t %t %d", s.PointA, s.PointB, s.Monotonic)
}

type velocity2State struct {
	// Histories contains histories for each window. A history contains previous single point values. Index 0 contains the latest, while the last contains the oldest value.
	Histories [][]velocity2Single
}

func newVelocity2State() *velocity2State {
	return &velocity2State{}
}

func (s *velocity2State) Shift(window int, single velocity2Single) {
	for i := len(s.Histories[window]) - 1; i > 0; i-- {
		s.Histories[window][i] = s.Histories[window][i-1]
	}
	s.Histories[window][0] = single
}

func sliceEqual(a, b []bool) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

type sensorPoint struct {
	ID    rune
	Point int64
}

func indexOrZero(s []bool, i int) bool {
	if len(s) <= i {
		return false
	}
	return s[i]
}

// position is where pointA is
// positive velocity is A → B
func Velocity2(breakbeam ActorRef, position int64) Actor {
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

	// NOTE: assume J-value does not change
	var sps []sensorPoint

	go func() {
		s := newVelocity2State()
		for d := range actor.InputCh {
			if d.Origin != breakbeam {
				panic(fmt.Sprintf("unknown origin %s", d.Origin))
			}
			now := time.Now()
			v := d.Value.(ValSeen)
			if sps == nil || s.Histories == nil {
				sps = make([]sensorPoint, 0, len(v.Sensors))
				for _, sensor := range v.Sensors {
					sps = append(sps, sensorPoint{
						ID:    rune(sensor.Name[0]),
						Point: sensor.Position,
					})
				}
				sort.Slice(sps, func(i, j int) bool { return sps[i].ID < sps[i].ID })
				s.Histories = make([][]velocity2Single, len(sps)-1)
				for window := 0; window < len(sps)-1; window++ {
					s.Histories[window] = make([]velocity2Single, 3)
				}
			}
			points := make([]bool, len(sps))
			for _, sensor := range v.Sensors {
				for i, sp := range sps {
					if sensor.Name == string(sp.ID) {
						points[i] = sensor.Seen
					}
				}
			}
			for window := 0; window < len(sps)-1; window++ {
				single := velocity2Single{
					Monotonic: v.Monotonic,
					PointA:    points[window],
					PointB:    points[window+1],
				}
				if len(s.Histories[0]) != 0 && s.Histories[window][0].PointA == single.PointA && s.Histories[window][0].PointB == single.PointB {
					// log.Printf("w%dequiv %v and %v", window, s.Histories[window][0], single)
					// equivalent to before
					continue
				}
				s.Shift(window, single)
				log.Printf("w%ds %v", window, s.Histories[window])
			}
			for window := 0; window < len(sps)-1; window++ {
				history := s.Histories[window]
				h0 := history[0]
				h0A := history[0].PointA
				h0B := history[0].PointB
				h1 := history[1]
				h1A := history[1].PointA
				h1B := history[1].PointB
				h2 := history[2]
				h2A := history[2].PointA
				h2B := history[2].PointB
				position := position + sps[window].Point
				interval := sps[window+1].Point - sps[window].Point
				// log.Printf("pos %v int %v", position, interval)
				// log.Printf("monotonic h0 %v h1 %v h2 %v", h0.Monotonic, h1.Monotonic, h2.Monotonic)

				a := ValAttitude{
					Monotonic: h0.Monotonic,
					Time:      now,
				}

				//   --> true direction/velocity means A → B
				//   A B
				// 1 x o (1 change before)
				// 0 x x (now)
				if h1A != h1B && h0A == false && h0B == false {
					a.Front = true
					dt := h0.Monotonic - h1.Monotonic
					// log.Printf("dt %v", dt)
					if dt != 0 {
						a.Velocity = interval * 1000 / dt
						if h1A {
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
						log.Printf("ATT1l w%d %s", window, a)
						actor.OutputCh <- Diffuse1{Value: a}
					}
				}
				//   A B
				// 2 x o
				// 1 x x
				// 0 o x
				if h2A != h2B && h1A == false && h1B == false && h0A != h0B && h0A != h2A {
					a.Front = false
					dt := h1.Monotonic - h2.Monotonic
					log.Printf("dt2 %v", dt)
					if dt != 0 {
						carsLength := interval*(h0.Monotonic-h1.Monotonic)/dt + interval
						a.Velocity = interval * 1000 / dt
						if h0A {
							// train is near point A + length now
							// A---B
							// >
							// ]====>
							//  ]====>
							a.Position = position + interval + carsLength
							log.Print("h0A")
						} else {
							// train is near point B - length now
							//   A---B
							//       <
							//   <====[
							// <====[
							a.Position = position + carsLength
							a.Velocity = -a.Velocity
							log.Print("!h0A")
						}
						log.Printf("ATT2l w%d %s cars%d pos%d", window, a, carsLength, a.Position)
						log.Printf("pos%d interval%d cars%d", position, interval, carsLength)
						actor.OutputCh <- Diffuse1{Value: a}
					}
				}
			}
		}
	}()
	return actor
}
