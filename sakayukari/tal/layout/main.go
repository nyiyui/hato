package layout

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"nyiyui.ca/hato/sakayukari/conn"
)

func reverse[S ~[]E, E any](s S) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

type LineID struct {
	Conn conn.Id
	// Usually A, B, C, or D.
	Line string
}

func (li LineID) String() string {
	return fmt.Sprintf("%s::%s", li.Conn, li.Line)
}

func (li *LineID) MarshalJSON() ([]byte, error) {
	return json.Marshal(li.String())
}

func (li *LineID) UnmarshalJSON(data []byte) error {
	inner := make([]byte, 0)
	err := json.Unmarshal(data, &inner)
	if err != nil {
		return err
	}
	parts := strings.SplitN(string(inner), "::", 2)
	li.Line = parts[1]
	li.Conn = conn.ParseId(parts[0])
	return nil
}

type Layout struct {
	Lines []Line
}

type Direction bool

// checkLinePort panics if pi doesn't exist in this Layout.
func (y *Layout) checkLinePort(pi LinePort) {
	if pi.LineI < 0 || pi.LineI >= len(y.Lines) {
		panic(fmt.Sprintf("invalid LinePort: LineI %d doesn't exist", pi.LineI))
	}
	if pi.PortI < 0 || pi.PortI > 2 {
		panic(fmt.Sprintf("invalid LinePort: PortI %d doesn't exist", pi.LineI))
	}
}

// MustLookup finds a lines with a matching ocmment. If it doesn't it panics.
// This is for debugging/testing.
func (y *Layout) MustLookup(comment string) Line {
	for _, l := range y.Lines {
		if l.Comment == comment {
			return l
		}
	}
	panic(fmt.Sprintf("found nothing when looking up for %s", comment))
}

// MustLookupIndex is MustLookup but returns an index.
func (y *Layout) MustLookupIndex(comment string) int {
	for li, l := range y.Lines {
		if l.Comment == comment {
			return li
		}
	}
	panic(fmt.Sprintf("found nothing when looking up for %s", comment))
}

func (y *Layout) Step(pi LinePort) (next LinePort, exists bool) {
	y.checkLinePort(pi)
	p := y.Lines[pi.LineI].GetPort(pi.PortI)
	if !p.notZero() || !p.ConnFilled {
		return LinePort{}, false
	}
	return LinePort{
		LineI: p.ConnI,
		PortI: p.ConnP,
	}, true
}

// LinePort contains an identitifer for a line and its port.
// This specifies both position and direction (a port is unidirectional, away from its own line).
type LinePort struct {
	LineI int
	PortI int
}

func (lp LinePort) String() string {
	return fmt.Sprintf("l%dp%d", lp.LineI, lp.PortI)
}

type Line struct {
	// Comment is a human-readable comment about the line.
	Comment string
	// PortA is the "base port", or one end of a piece of track. All other ports branch from here, so e.g. a switch's merged end must be at a base port.
	PortA Port
	// PortB is the normal side of a switch, or the other end of a straight track.
	PortB Port
	// PortC is the reverse side of a switch.
	PortC Port
	// PowerConn is the connection and line ID for the soyuu-line hardware controlling this line's power.
	// One LineID can correspond to one Line's power.
	PowerConn LineID
	// SwitchConn is the connection and line ID for the soyuu-line hardware controlling this line's switch.
	// One LineID can correspond to one Line's switch.
	// The A direction sets the switch to the normal position, and the B direction sets the switch to the reverse position.
	SwitchConn LineID
}

func (l Line) IsSwitch() bool {
	if l.SwitchConn != (LineID{}) != l.PortC.notZero() {
		panic("SwitchConn and PortC initialisation mismatch")
	}
	return l.SwitchConn != (LineID{})
}

func StraightLine(length uint32) Line {
	return Line{
		PortA: Port{Length: 0},
		PortB: Port{Length: length},
	}
}

func Turnout(lengthA, lengthB uint32, reverse []Line) Line {
	return Line{
		PortA: Port{Length: 0},
		PortB: Port{Length: lengthA},
		PortC: Port{Length: lengthB, ConnInline: reverse},
	}
}

func (l *Line) GetPort(p int) Port {
	if p == 0 {
		return l.PortA
	}
	if p == 1 {
		return l.PortB
	}
	if p == 2 {
		return l.PortC
	}
	panic(fmt.Sprintf("unknown port %d", p))
}

func Connect(lines []Line) (Layout, error) {
	y := Layout{Lines: make([]Line, 0, len(lines))}
	err := y.connect(lines)
	return y, err
}

func (y *Layout) connect(lines []Line) error {
	originalLen := len(y.Lines)
	// TODO: test ConnInline (layouts with switches)
	// only do prev→next line connections here; next→prev conns are added later
	for li2, l := range lines {
		i := len(y.Lines) - originalLen
		if len(l.PortC.ConnInline) != 0 {
			li := len(y.Lines)
			y.Lines = append(y.Lines, l)
			inlineLen := len(l.PortC.ConnInline)
			err := y.connect(l.PortC.ConnInline)
			if err != nil {
				return fmt.Errorf("line %d PortC inline: %w", li, err)
			}
			y.Lines[li].PortB.ConnI = li + 1 + inlineLen
			y.Lines[li].PortB.ConnP = 0
			y.Lines[li].PortB.ConnFilled = true
			y.Lines[li].PortC.ConnI = i + 1
			y.Lines[li].PortC.ConnP = 0
			y.Lines[li].PortC.ConnFilled = true
			y.Lines[li].PortC.ConnInline = nil
			// TODO: last Line of ConnInline is not ConnI:-1
			if i != 0 {
				y.Lines[li-1].PortB.ConnI = li
				y.Lines[li-1].PortB.ConnP = 0
				y.Lines[li-1].PortB.ConnFilled = true
			}
			i += inlineLen
		} else if l.PortB.notZero() {
			y.Lines = append(y.Lines, l)
			if i != 0 && y.Lines[i-1].PortB.ConnI != -1 {
				y.Lines[li2-1].PortB.ConnI = i
				y.Lines[li2-1].PortB.ConnP = 0
				y.Lines[li2-1].PortB.ConnFilled = true
			}
		} else {
			return fmt.Errorf("unsupported line %d: %#v", li2, l)
		}
		i++
	}
	li := len(y.Lines) - 1
	y.Lines[li].PortB.ConnI = -1
	y.Lines[li].PortB.ConnP = -1
	y.Lines[li].PortB.ConnFilled = false
	return y.connectFix()
}

func (y *Layout) connectFix() error {
	for li, _ := range y.Lines {
		for p := 0; p <= 2; p++ {
			var j, q int
			if p == 0 {
				j = y.Lines[li].PortA.ConnI
				q = y.Lines[li].PortA.ConnP
				if !y.Lines[li].PortA.ConnFilled {
					continue
				}
			} else if p == 1 {
				j = y.Lines[li].PortB.ConnI
				q = y.Lines[li].PortB.ConnP
				if !y.Lines[li].PortB.ConnFilled {
					continue
				}
			} else if p == 2 {
				j = y.Lines[li].PortC.ConnI
				q = y.Lines[li].PortC.ConnP
				if !y.Lines[li].PortC.ConnFilled {
					continue
				}
			}
			if q == 0 {
				y.Lines[j].PortA.ConnI = li
				y.Lines[j].PortA.ConnP = 1
				y.Lines[j].PortA.ConnFilled = true
			} else if q == 1 {
				y.Lines[j].PortB.ConnI = li
				y.Lines[j].PortB.ConnP = 1
				y.Lines[j].PortB.ConnFilled = true
			} else if q == 2 {
				y.Lines[j].PortC.ConnI = li
				y.Lines[j].PortC.ConnP = 1
				y.Lines[j].PortC.ConnFilled = true
			}
		}
	}
	return nil
}

// Port is a connection point from a line to another line.
// For example, a switch would have 3 ports: the base port, and 2 additional ports.
type Port struct {
	// Length from the base port (in µm). Must be 0 for a base port.
	// TODO: replace this a more flexible way to describe shape (this can only describe a straight line)
	Length uint32
	// ConnFilled must be true to use ConnI and ConnP.
	ConnFilled bool
	// ConnI is the index of the line this connects to in the layout. Set to -1 if there is no connection.
	ConnI int
	// ConnI is the port of the line this connects to in the layout. Set to -1 if there is no connection.
	ConnP int
	// ConnInline is the line for the connection.
	ConnInline []Line
	// TODO: how to represent curves?
	// Direction is the direction power must be set at to make a train move towards this port.
	Direction bool
}

func (p Port) String() string {
	if p.ConnFilled {
		return fmt.Sprintf("l%dµm → i%d/p%d (%#v)", p.Length, p.ConnI, p.ConnP, p.ConnInline)
	} else {
		return fmt.Sprintf("l%dµm → NA (%#v)", p.Length, p.ConnInline)
	}
}

func (p *Port) notZero() bool {
	return p.Length != 0 || p.ConnFilled || p.ConnI != 0 || p.ConnP != 0 || p.ConnInline != nil
}

// countLength sums the length of the total layout. The layout must not contain switches.
func (y *Layout) countLength() uint32 {
	i := 0
	p := 1
	var sum uint32 = 0
	for {
		l := y.Lines[i]
		port := l.GetPort(p)
		sum += port.Length
		if port.ConnFilled {
			i = port.ConnI
			p = port.ConnP
		} else {
			return sum
		}
	}
	return sum
}

// PathTo returns a list of outgoing LinePorts in the order they should be followed.
// This assumes all switches can be operated in both normal and reverse directions.
func (y *Layout) PathTo(from, goal int) []LinePort {
	// simple Dijkstra's, using the "using" slice to track the shortest path
	const infinity = -1
	const debug = false
	if from == goal {
		return nil
	}
	visited := make([]bool, len(y.Lines))
	distance := make([]int, len(y.Lines))
	using := make([]LinePort, len(y.Lines))
	for i := range distance {
		if i == from {
			continue
		}
		distance[i] = infinity
	}
	queue := make([]LinePort, 0, len(y.Lines))
	queue = append(queue, LinePort{from, 1}, LinePort{from, 2})
	for current := (LinePort{from, 0}); len(queue) > 0; current, queue = queue[0], queue[1:] {
		if debug {
			log.Print("---")
			log.Printf("current %#v", current)
			log.Printf("queue %#v", queue)
		}
		l := y.Lines[current.LineI]
		for i := 0; i <= 2; i++ {
			if debug {
				log.Printf("port %d", i)
			}
			p := l.GetPort(i)
			if !p.ConnFilled {
				if debug {
					log.Printf("unfilled")
				}
				continue
			}
			if i+current.PortI == 5 {
				// cannot go between ports B and C
				if debug {
					log.Printf("between")
				}
				continue
			}
			if distance[p.ConnI] == infinity || distance[current.LineI] < distance[p.ConnI] {
				distance[p.ConnI] = distance[current.LineI] + 1
				using[p.ConnI] = LinePort{current.LineI, i}
				queue = append(queue, LinePort{p.ConnI, i})
				if debug {
					log.Printf("add %#v", queue[len(queue)-1])
				}
			}
		}
		visited[current.LineI] = true
		if distance[goal] != infinity {
			break
		}
	}
	log.Printf("distance[%d]: %#v", goal, distance)
	lps := make([]LinePort, distance[goal])
	for i, j := goal, 0; i != from; i, j = using[i].LineI, j+1 {
		lps[len(lps)-1-j] = using[i]
	}
	return lps
}
