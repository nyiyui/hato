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
	Points    []bool
}

func (s *velocity2Single) String() string {
	return fmt.Sprintf("%#v %d", s.Points, s.Monotonic)
}

type velocity2State struct {
	// History contains previous single point values. Index 0 contains the latest, while the last contains the oldest value.
	History []velocity2Single
}

func newVelocity2State() *velocity2State {
	return &velocity2State{
		History: make([]velocity2Single, 16), // TODO: History slices per window (to have an upper limit on memory) (length was arbitrarily chosen)
	}
}

func (s *velocity2State) Shift(single velocity2Single) {
	for i := len(s.History) - 1; i > 0; i-- {
		s.History[i] = s.History[i-1]
	}
	s.History[0] = single
}

// GetHistory returns the oldest history entry for index i that has changes (w.r.t. prev entry) to pointA and pointB
func (s *velocity2State) GetHistory(pointA, pointB, i int) velocity2Single {
	wh := make([]velocity2Single, 0, 3) // windowed history
	for i := 0; i < len(s.History)-1; i++ {
		prev := s.History[i+1]
		cur := s.History[i]
		if cur.Points[pointA] == prev.Points[pointA] && cur.Points[pointB] == prev.Points[pointB] {
			fmt.Printf("%d cur %#v\n", i, cur)
			fmt.Printf("%d prev %#v\n", i, prev)
			fmt.Printf("%d continue\n", i)
			continue
		}
		if len(wh) != 0 {
			last := wh[len(wh)-1]
			if cur.Points[pointA] == last.Points[pointA] && cur.Points[pointB] == last.Points[pointB] {
				// make sure it's the oldest only
				wh[len(wh)-1] = cur
			} else {
				wh = append(wh, cur)
			}
		} else {
			wh = append(wh, cur)
		}
	}
	wh = append(wh, s.History[len(s.History)-1])
	fmt.Printf("wh %#v\n", wh)
	return wh[i]
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
	ActorLoop:
		for d := range actor.InputCh {
			if d.Origin != breakbeam {
				panic(fmt.Sprintf("unknown origin %s", d.Origin))
			}
			now := time.Now()
			v := d.Value.(ValSeen)
			if sps == nil {
				sps = make([]sensorPoint, 0, len(v.Sensors))
				for _, sensor := range v.Sensors {
					sps = append(sps, sensorPoint{
						ID:    rune(sensor.Name[0]),
						Point: sensor.Position,
					})
				}
				sort.Slice(sps, func(i, j int) bool { return sps[i].ID < sps[i].ID })
			}
			single := velocity2Single{
				Monotonic: v.Monotonic,
				Points:    make([]bool, len(sps)),
			}
			for _, sensor := range v.Sensors {
				for i, sp := range sps {
					if sensor.Name == string(sp.ID) {
						single.Points[i] = sensor.Seen
					}
				}
			}
			if sliceEqual(s.History[0].Points, single.Points) {
				// equivalent to before
				continue ActorLoop
			}
			s.Shift(single)
			log.Printf("s %v", s.History)
			for window := 0; window < len(sps)-1; window++ {
				h0 := single
				h0A := single.Points[window]
				h0B := single.Points[window+1]
				h1 := s.History[1]
				h1A := indexOrZero(s.History[1].Points, window)
				h1B := indexOrZero(s.History[1].Points, window+1)
				h2 := s.History[2]
				h2A := indexOrZero(s.History[2].Points, window)
				h2B := indexOrZero(s.History[2].Points, window+1)
				position := position + sps[window].Point
				interval := sps[window].Point - sps[window+1].Point
				log.Printf("pos %v int %v", position, interval)
				log.Printf("monotonic h0 %v h1 %v h2 %v", h0.Monotonic, h1.Monotonic, h2.Monotonic)

				a := ValAttitude{
					Monotonic: h0.Monotonic,
					Time:      now,
				}

				// === Longer-than-interval cars

				//   --> true direction/velocity means A → B
				//   A B
				// 1 x o (1 change before)
				// 0 x x (now)
				if h1A != h1B && h0A == false && h0B == false {
					a.Front = true
					dt := h0.Monotonic - h1.Monotonic
					log.Printf("dt %v", dt)
					if dt != 0 {
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
						a.Velocity = interval * 1000 / dt
						log.Printf("att1l %s", a)
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
						if h0A {
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
					if h2A != h2B && h1A == true && h1B == true && h0A != h0B {
						a.Front = true
						dt := h2.Monotonic - h0.Monotonic
						if dt != 0 {
							a.Velocity = interval * 1000 / dt
							if h0A {
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
		}
	}()
	return actor
}
