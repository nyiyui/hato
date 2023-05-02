package conn

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"sync"

	"nyiyui.ca/soyuu/soyuuctl/sakayukari"
)

type Actor = sakayukari.Actor
type Value = sakayukari.Value

// TypeHandler handles a type instance.
type TypeHandler func(s *State, path string, f io.ReadWriter, c *Conn)

var typeHandlers = map[string]TypeHandler{
	"soyuu-line-mega-0": handleLine,
	// "soyuu-dist":        handleDist,
	"soyuu-breakbeam": handleBreakbeam,
}

type Path = string
type ConnName = string
type LineName = string
type STName = string // ST = 照査点

type State struct {
	conns      map[ConnName]*Conn
	connsLock  sync.RWMutex
	notifyNew  chan string
	actors     map[string]*sakayukari.Actor
	actorsLock sync.Mutex
	SetupDone  sync.WaitGroup
}

func NewState() *State {
	return &State{
		conns:     map[ConnName]*Conn{},
		notifyNew: make(chan string),
		actors:    map[string]*sakayukari.Actor{},
	}
}

func (s *State) Actors() map[string]*sakayukari.Actor { return s.actors }
func (s *State) Find() error                          { return s.find() }

func (s *State) GetConn(path ConnName) (*Conn, bool) {
	s.connsLock.RLock()
	defer s.connsLock.RUnlock()
	c, ok := s.conns[path]
	if !ok {
		return nil, false
	}
	return c, true
}

type Conn struct {
	Id Id
}

type Req interface {
	isReq()
}

type ReqLine struct {
	Line      LineName
	Brake     bool
	Direction bool
	Power     uint8
}

func (_ ReqLine) isReq() {}

func (r ReqLine) String() string {
	var send [7]byte
	// CAAN000
	// C - change
	//  A - line
	//   A - direction
	//    N - brake
	//     000 - power
	send[0] = 'C'
	send[1] = r.Line[0]
	if r.Direction {
		send[2] = 'A'
	} else {
		send[2] = 'B'
	}
	if r.Brake {
		send[3] = 'Y'
	} else {
		send[3] = 'N'
	}
	power := fmt.Sprintf("%03d", r.Power)
	copy(send[4:], power)
	return string(send[:])
}

type Id struct {
	Type     string
	Variant  string
	Instance string
}

func (i Id) String() string {
	return fmt.Sprintf("%s/%s-%s", i.Type, i.Variant, i.Instance)
}

func parseId(id string) Id {
	ss := strings.Split(id, "/")
	if len(ss) < 3 {
		ss = append(ss, "")
		ss = append(ss, "")
	}
	// soyuu-breakbeam/itsybitsy0/0
	id2 := Id{
		Type:     ss[0],
		Variant:  ss[1],
		Instance: ss[2],
	}
	return id2
}

func (s *State) handleConn(path string, f io.ReadWriter, c *Conn) {
	handler := typeHandlers[c.Id.Type]
	if handler == nil {
		log.Printf("no handler found for %s %s", path, c.Id)
		return
	}
	log.Printf("handling %s %s", path, c.Id)
	handler(s, path, f, c)
}

type lineValue struct {
	Line  string
	Value Value
}

func handleLine(s *State, path string, f io.ReadWriter, c *Conn) {
	names := []string{"A", "B", "C", "D"}

	updates := make(chan lineValue, 0)
	// Actor's DependsOn is filled by another function
	lines := map[string]*Actor{}
	for _, name := range names {
		name := name
		lines[name] = &Actor{
			UpdateFunc: func(self *Actor, gs sakayukari.GraphState) Value {
				if len(self.DependsOn) == 0 {
					log.Printf("%s: len is 0", name)
					return nil
				}
				updates <- lineValue{
					Line:  name,
					Value: gs.States[self.DependsOn[0]],
				}
				return nil
			},
			SideEffects: true,
		}
	}
	func() {
		s.actorsLock.Lock()
		defer s.actorsLock.Unlock()
		for name, actor := range lines {
			s.actors[c.Id.String()+":"+name] = actor
		}
	}()
	s.SetupDone.Done()

	for lv := range updates {
		switch req := lv.Value.(type) {
		case ReqLine:
			req.Line = lv.Line
			_, err := fmt.Fprintf(f, "%s\n", req.String())
			if err != nil {
				log.Printf("commit %s: %s", req, err)
			}
		default:
			log.Printf("invalid req received: %+v", req)
		}
	}
}

/*
func handleDist(s *State, path string, f io.ReadWriter, c *Conn) {
	func() {
		s.stsLock.Lock()
		defer s.stsLock.Unlock()
		s.sts[c.Id.Instance] = path
	}()
	close(c.Reqs)
	var curState STState
	var curPos, prevPos int64
	var curVel int64
	var curVelValid bool
	_ = curVelValid
	var curTS, prevTS int64
	log.Printf("===set GetValue")
	reader := bufio.NewReader(f)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("%s: read line: %s", path, err)
			continue
		}
		if !strings.HasPrefix(line, " S") {
			continue
		}
		line = strings.TrimSpace(line[2:])
		parts := strings.SplitN(line, " ", 3)
		state, err := strconv.ParseInt(parts[0], 10, 32)
		if err != nil {
			log.Printf("%s: parse state: %s", path, err)
			continue
		}
		curState = STState(state)
		curPos, err = strconv.ParseInt(parts[1], 10, 32)
		if err != nil {
			log.Printf("%s: parse pos: %s", path, err)
			continue
		}
		curTS, err = strconv.ParseInt(parts[2], 10, 32)
		if err != nil {
			log.Printf("%s: parse now: %s", path, err)
			continue
		}
		if curState == STStateSide && curTS != prevTS {
			curVel = (curPos - prevPos) / (curTS - prevTS)
			curVelValid = true
			log.Printf("1 %s: state %v pos %v → %v ts %v → %v vel %v of %s", path, curState, prevPos, curPos, prevTS, curTS, curVel, line)
			prevPos = curPos
			prevTS = curTS
		} else {
			curVelValid = false
		}
		log.Print(curPos)
		v := ValAttitude{
			State:     curState,
			Position:  curPos,
			Velocity:  curVel,
			Monotonic: curTS,
			Certain:   true,
		}
		c.Actor.GetChan <- v
	}
}
*/

func handleBreakbeam(s *State, path string, f io.ReadWriter, c *Conn) {
	beamNames := []string{"A", "B"}
	beams := map[string]*Actor{}
	for _, name := range beamNames {
		beams[name] = &Actor{
			RecvChan:    make(chan Value),
			SideEffects: true,
		}
	}
	func() {
		s.actorsLock.Lock()
		defer s.actorsLock.Unlock()
		for _, name := range beamNames {
			s.actors[c.Id.String()+":"+name] = beams[name]
		}
	}()
	s.SetupDone.Done()
	reader := bufio.NewReader(f)
	fmt.Fprint(f, "Si0100\n")
	log.Printf("breakbeam: setup done %s %#v", path, c)
	prevValues := map[string]ValSeen{}
	prevDisableUntil := int64(0)
ReadLoop:
	for {
		lineRaw, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("%s: read line: %s", path, err)
			continue
		}
		if !strings.HasPrefix(lineRaw, " D") {
			continue
		}
		// TODO: adjust log interval based on value
		line := lineRaw[2:]
		var monotonic int64
		values := map[string]bool{}
		for i := 0; i < len(line); i++ {
			switch {
			case line[i] == 'T':
				j := strings.IndexFunc(line[i+1:], func(r rune) bool { return r >= 'A' && r <= 'Z' })
				if j == -1 {
					j = len(line) - i
				}
				monotonic, err = strconv.ParseInt(strings.TrimSpace(line[i+1:i+j]), 10, 64)
				if err != nil {
					log.Printf("parse: T: %s", err)
					continue ReadLoop
				}
				i += j
			default:
				values[string(line[i])] = line[i+1] == '1'
				i++
			}
		}
		if prev, ok := prevValues["A"]; ok && monotonic > prevDisableUntil {
			delta := monotonic - prev.Monotonic
			if prev.Seen != values["B"] {
				velocity := 248 * 1000 * 1000 / delta // µm/s
				_ = velocity
				//log.Printf("velocity %d µm/s", velocity)
				prevDisableUntil = monotonic + 1000
			}
		}
		for sensor, value := range values {
			v := ValSeen{
				Monotonic: monotonic,
				Sensor:    sensor,
				Seen:      value,
			}
			//log.Printf("sense %s %t", sensor, value)
			beamActor, ok := beams[sensor]
			if !ok {
				panic("sensor's actor missing")
			}
			beamActor.RecvChan <- v
			if prev, ok := prevValues[sensor]; !ok || prev.Seen != value {
				prevValues[sensor] = v
			}
		}
	}
}
