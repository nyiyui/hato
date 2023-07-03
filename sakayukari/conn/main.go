package conn

import (
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"time"

	. "nyiyui.ca/hato/sakayukari"
)

func ConnActors(expected []Id) (*State, []Actor) {
	s := new(State)
	s.conns = map[ConnName]*Conn{}
	s.actors = map[Id]Actor{}

	as := make([]Actor, 0, len(expected))
	for _, id := range expected {
		a := handlers[id.Type].NewBlankActor()
		as = append(as, a)
		s.actors[id] = a
	}
	return s, as
}

type Handler interface {
	HandleConn(a Actor, c *Conn)
	NewBlankActor() Actor
}

var handlers = map[string]Handler{
	"soyuu-line":      handlerLine{},
	"soyuu-breakbeam": handlerBreakbeam{},
	"soyuu-rfid":      handlerRFID{},
}

type Path = string
type ConnName = string
type LineName = string
type STName = string // ST = 照査点

type State struct {
	conns     map[ConnName]*Conn
	connsLock sync.RWMutex
	actors    map[Id]Actor
}

func (s *State) Find() error { return s.find() }

type Conn struct {
	Id   Id
	Path string
	F    io.ReadWriter
}

type ReqSwitch struct {
	Line       LineName
	BrakeAfter bool
	Direction  bool
	Power      uint8
	Timeout    time.Duration
}

func (r ReqSwitch) String() string {
	opts := ""
	if r.BrakeAfter {
		opts += "b"
	}
	if r.Direction {
		opts += "D"
	} else {
		opts += "d"
	}
	return fmt.Sprintf("switch(%s %s %d %s)", r.Line, opts, r.Power, r.Timeout)
}

func AbsClampPower(power int) uint8 {
	if power < 0 {
		power *= -1
	}
	if power > 255 {
		power = 255
	}
	return uint8(power)
}

type ReqLines struct {
	ReqLines []ReqLine
}

func (r ReqLines) String() string {
	b := new(strings.Builder)
	for _, rl := range r.ReqLines {
		fmt.Fprintf(b, "%s\n", rl)
	}
	return b.String()
}

type ReqLine struct {
	Line      LineName
	Brake     bool
	Direction bool
	Power     uint8
}

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

func (s *State) handleConn(c *Conn) {
	handler := handlers[c.Id.Type]
	if handler == nil {
		log.Printf("no handler found for %s %s", c.Path, c.Id)
		return
	}
	log.Printf("handling %s %s", c.Path, c.Id)
	handler.HandleConn(s.actors[c.Id], c)
}
