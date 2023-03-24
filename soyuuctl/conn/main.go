package conn

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"sync"
)

// TypeHandler handles a type instance.
type TypeHandler func(s *State, path string, f io.ReadWriter, c *Conn)

var typeHandlers = map[string]TypeHandler{
	"soyuu-line-mega-0": handleLine,
	"soyuu-dist":        handleDist,
	"soyuu-breakbeam":   handleBreakbeam,
}

type Path = string
type ConnName = string
type LineName = string
type STName = string // ST = 照査点

type State struct {
	conns     map[ConnName]*Conn
	connsLock sync.RWMutex
	notifyNew chan string
	lines     map[LineName]ConnName
	linesLock sync.RWMutex
	sts       map[STName]ConnName
	stsLock   sync.RWMutex
}

func NewState() *State {
	return &State{
		conns:     map[ConnName]*Conn{},
		notifyNew: make(chan string),
		lines:     map[LineName]ConnName{},
		sts:       map[STName]ConnName{},
	}
}

func (s *State) Find() error {
	return s.find()
}

func (s *State) Req(path ConnName, req Req) error {
	s.connsLock.RLock()
	defer s.connsLock.RUnlock()
	c, ok := s.conns[path]
	if !ok {
		return errors.New("conn not found")
	}
	c.Reqs <- req
	return nil
}

func (s *State) GetST(name STName) (*Conn, bool) {
	s.stsLock.RLock()
	defer s.stsLock.RUnlock()
	connName, ok := s.sts[name]
	if !ok {
		return nil, false
	}
	c, ok := s.conns[connName]
	if !ok {
		panic("conn not found (stale st!)")
	}
	return c, true
}

func (s *State) GetConn(path ConnName) (*Conn, bool) {
	s.connsLock.RLock()
	defer s.connsLock.RUnlock()
	c, ok := s.conns[path]
	if !ok {
		return nil, false
	}
	return c, true
}

func (s *State) LineReq(line LineName, req ReqLine) error {
	s.linesLock.RLock()
	defer s.linesLock.RUnlock()
	path := s.lines[line]
	req.Line = line
	return s.Req(path, req)
}

type Conn struct {
	Id        Id
	Reqs      chan Req
	HooksLock sync.Mutex
	Hooks     []func(v Val)
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

type Id struct {
	Type     string
	Variant  string
	Instance string
}

func (i Id) String() string {
	return fmt.Sprintf("%s/%s-%s", i.Type, i.Variant, i.Instance)
}

func parseId(id string) Id {
	ss := strings.Split(id, " ")
	tv := ss[0]
	if len(ss) < 2 {
		ss = append(ss, "")
	}
	tv2 := strings.SplitN(tv, "/", 2)
	if len(tv2) < 2 {
		tv2 = append(tv2, "")
	}
	return Id{
		Type:     tv2[0],
		Variant:  tv2[1],
		Instance: ss[1],
	}
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

func handleLine(s *State, path string, f io.ReadWriter, c *Conn) {
	func() {
		s.linesLock.Lock()
		defer s.linesLock.Unlock()
		s.lines["A"] = path
		s.lines["B"] = path
		s.lines["C"] = path
		s.lines["D"] = path
	}()
	for req := range c.Reqs {
		switch req := req.(type) {
		case ReqLine:
			var send [7]byte
			// CAAN000
			// C - change
			//  A - line
			//   A - direction
			//    N - brake
			//     000 - power
			send[0] = 'C'
			send[1] = req.Line[0]
			if req.Direction {
				send[2] = 'A'
			} else {
				send[2] = 'B'
			}
			if req.Brake {
				send[3] = 'Y'
			} else {
				send[3] = 'N'
			}
			power := fmt.Sprintf("%03d", req.Power)
			copy(send[4:], power)
			_, err := fmt.Fprintf(f, "%s\n", send)
			if err != nil {
				log.Printf("commit %s: %s", send, err)
			}
		default:
			log.Print("invalid req received")
		}
	}
}

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
		func() {
			c.HooksLock.Lock()
			defer c.HooksLock.Unlock()
			for _, hook := range c.Hooks {
				hook(v)
			}
		}()
	}
}

func handleBreakbeam(s *State, path string, f io.ReadWriter, c *Conn) {
	func() {
		s.stsLock.Lock()
		defer s.stsLock.Unlock()
		s.sts[c.Id.Instance+"A"] = path
		s.sts[c.Id.Instance+"B"] = path
	}()
	close(c.Reqs)
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
		line = lineRaw[2:]
		var monotonic int64
		var values map[string]bool
		var err error
		for i := 0; i < len(line); i++ {
			switch {
			case line[i] == "T":
				j := strings.IndexFunc(line[i:], func(r rune) bool { return r >= "A" && r <= "Z" })
				monotonic, err = strconv.ParseInt(line[i:i+j], 10, 64)
				if err != nil {
					log.Printf("parse: T: %s", err)
					continue
				}
				i += j
			default:
				j := strings.IndexFunc(line[i:], func(r rune) bool { return r >= "A" && r <= "Z" })
				values[string(line[i])] = line[i+1] == "1"
				i += j
			}
		}
		for sensor, value := range values {
			v := ValSeen{
				Monotonic: monotonic,
				Sensor:    sensor,
				Seen:      value,
			}
			func() {
				c.HooksLock.Lock()
				defer c.HooksLock.Unlock()
				for _, hook := range c.Hooks {
					hook(v)
				}
			}()
		}
	}
}
