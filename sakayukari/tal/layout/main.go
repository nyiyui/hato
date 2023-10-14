package layout

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"golang.org/x/exp/slices"
	"nyiyui.ca/hato/sakayukari/conn"
)

// Length represents a length in 1µm precision.
type Length = int64

const (
	Millimeter = 1000
	Micrometer = 1
)

// LineI is a line index, an index of a slice with Lines.
type LineI int

// BlankLineI is used as a null value for LineI (0 has a meaning).
const BlankLineI = -123

// PortI is a port index, representing ports A, B, and C.
type PortI int

func (p PortI) String() string {
	switch p {
	case 0:
		return "A"
	case 1:
		return "B"
	case 2:
		return "C"
	default:
		return strconv.FormatInt(int64(p), 10)
	}
}

const (
	PortDNC PortI = -1
	// use non 0-3 numbers to error out on legacy code
	PortA PortI = 0
	PortB PortI = 1
	PortC PortI = 2
)

func reverse[S ~[]E, E any](s S) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

type RFID struct {
	Conn    conn.Id
	Precise int64
}

type LineID struct {
	Conn conn.Id
	// Usually A, B, C, or D.
	Line string
}

func (li LineID) String() string {
	return fmt.Sprintf("%s/%s/%s::%s", li.Conn.Type, li.Conn.Variant, li.Conn.Instance, li.Line)
}

func (li LineID) MarshalJSON() ([]byte, error) {
	return json.Marshal(li.String())
}

func (li *LineID) UnmarshalJSON(data []byte) error {
	var inner string
	err := json.Unmarshal(data, &inner)
	if err != nil {
		return err
	}
	parts := strings.SplitN(inner, "::", 2)
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
	if pi.LineI < 0 || int(pi.LineI) >= len(y.Lines) {
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
func (y *Layout) MustLookupIndex(comment string) LineI {
	for li, l := range y.Lines {
		if l.Comment == comment {
			return LineI(li)
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
	LineI LineI
	PortI PortI
}

func (lp LinePort) String() string {
	return fmt.Sprintf("%d%s", lp.LineI, lp.PortI)
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
	RFIDs      []RFID
}

func (l Line) IsSwitch() bool {
	if l.SwitchConn != (LineID{}) != l.PortC.notZero() {
		panic("SwitchConn and PortC initialisation mismatch - SwitchConn should be defined if PortC is defined!")
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

func (l *Line) GetPort(p PortI) Port {
	if p == PortA {
		return l.PortA
	}
	if p == PortB {
		return l.PortB
	}
	if p == PortC {
		return l.PortC
	}
	panic(fmt.Sprintf("unknown port %d", p))
}

func (l *Line) SetPort(pi PortI, p Port) {
	if pi == PortA {
		l.PortA = p
	} else if pi == PortB {
		l.PortB = p
	} else if pi == PortC {
		l.PortC = p
	} else {
		panic(fmt.Sprintf("unknown port %d", pi))
	}
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
		i := LineI(len(y.Lines) - originalLen)
		if len(l.PortC.ConnInline) != 0 {
			li := LineI(len(y.Lines))
			y.Lines = append(y.Lines, l)
			inlineLen := LineI(len(l.PortC.ConnInline))
			err := y.connect(l.PortC.ConnInline)
			if err != nil {
				return fmt.Errorf("line %d PortC inline: %w", li, err)
			}
			y.Lines[li].PortB.ConnI = li + 1 + inlineLen
			y.Lines[li].PortB.nerfOutOfRangeConn = true
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
		} else if len(l.PortB.ConnInline) != 0 {
			return fmt.Errorf("line %d PortB cannot have ConnInline", li2)
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
	return y.connectTransposed()
}

func (y *Layout) connectTransposed() error {
	//data, _ := json.MarshalIndent(y, "", "  ")
	//log.Printf("connectTransposed json: %s", data)
	for li, _ := range y.Lines {
		for pi := PortI(0); pi <= 2; pi++ {
			p := y.Lines[li].GetPort(pi)
			if p.nerfOutOfRangeConn && int(p.ConnI) >= len(y.Lines) {
				p.ConnI = -1
				p.ConnP = -1
				p.ConnFilled = false
				l := y.Lines[li]
				l.SetPort(pi, p)
				y.Lines[li] = l
			}
			if !p.ConnFilled {
				continue
			}
			p2 := y.Lines[p.ConnI].GetPort(p.ConnP)
			p2.ConnI = LineI(li)
			p2.ConnP = pi
			p2.ConnFilled = true
			l2 := y.Lines[p.ConnI]
			l2.SetPort(p.ConnP, p2)
			y.Lines[p.ConnI] = l2
		}
	}
	return nil
}

// Port is a connection point from a line to another line.
// For example, a switch would have 3 ports: the base port, and 2 additional ports.
type Port struct {
	// Length from the base port (in µm). Must be 0 for a base port, and must not be 0 for a non-base port.
	// TODO: replace this a more flexible way to describe shape (this can only describe a straight line)
	Length uint32
	// ConnFilled must be true to use ConnI and ConnP.
	ConnFilled bool
	// ConnI is the index of the line this connects to in the layout. Set to -1 if there is no connection.
	ConnI LineI
	// ConnI is the port of the line this connects to in the layout. Set to -1 if there is no connection.
	ConnP PortI
	// ConnInline is the line for the connection.
	ConnInline []Line
	// TODO: how to represent curves?
	// Direction is the direction power must be set at to make a train move towards this port.
	Direction          bool
	nerfOutOfRangeConn bool
}

func (p Port) Conn() LinePort {
	return LinePort{p.ConnI, p.ConnP}
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

//// Measure returns the distance from the first LinePort to the last LinePort.
//func (y *Layout) Measure(path []LinePort) int64 {
//	panic("not implemented yet")
//}

/*
// reversePath "reverses" the path given.
// The starting location is not preserved.
func (y *Layout) reversePath(path []LinePort) []LinePort {
	reversed := make([]LinePort, len(path))
	j := len(reversed) - 1
	for pathI := range path {
		cur := path[pathI]
		if pathI == 0 {
			continue
		} else {
			prev := path[pathI-1]
			p := y.Lines[prev.LineI].GetPort(prev.PortI)
			if p.ConnI != cur.LineI {
				panic("path not connected")
			}
			reversed[j] = LinePort{p.ConnI, p.ConnP}
			j--
		}
	}
	return reversed
}
*/

func (y *Layout) GetLinePort(lp LinePort) (Line, Port) {
	l := y.Lines[lp.LineI]
	return l, l.GetPort(lp.PortI)
}

func (y *Layout) GetPort(lp LinePort) Port {
	l := y.Lines[lp.LineI]
	return l.GetPort(lp.PortI)
}

// Count returns the distance between start and end, going through the path specified.
// The first LineI of path must match start.
func (y *Layout) Count(path []LinePort, start, end Position) (dist int64) {
	if path[0].LineI != start.LineI {
		panic("First LinePort of path doesn't match start.")
	}
	if path[len(path)-1].LineI != end.LineI {
		panic("Last LinePort of path doesn't match end.")
	}
	if path[len(path)-1].PortI == -1 {
		panic("Last LinePort should not be one from PathToInclusive, it should be the same as all non-last LinePorts.")
	}
	for pathI, lp := range path {
		if pathI == 0 {
			switch lp.PortI {
			case PortA:
				dist += int64(start.Precise)
			case PortB, PortC:
				_, p := y.GetLinePort(lp)
				dist += int64(p.Length - start.Precise)
			default:
				panic("invalid port")
			}
		} else {
			var length uint32
			prevLP := path[pathI-1]
			if prevLP.LineI == lp.LineI {
				panic("Count cannot handle paths that contain a starting point (two consecutive LinePorts with the same LineI) (e.g. those generated by FullPathTo)")
			}
			_, prevP := y.GetLinePort(prevLP)
			if prevP.ConnI != lp.LineI {
				log.Printf("prevLP %#v", prevLP)
				log.Printf("prevP %#v", prevP)
				log.Printf("lp %#v", lp)
				panic("path is not connected")
			}
			l := y.Lines[lp.LineI]
			switch prevP.ConnP {
			case PortA:
				length = l.GetPort(lp.PortI).Length
			case PortB, PortC:
				if lp.PortI == PortB || lp.PortI == PortC {
					panic("cannot go between ports B and C")
				}
				length = l.GetPort(prevP.ConnP).Length
			default:
				panic("invalid port")
			}
			dist += int64(length)
		}
		if pathI == len(path)-1 {
			lp := path[pathI]
			switch lp.PortI {
			case PortA:
				dist -= int64(end.Precise)
			case PortB, PortC:
				_, p := y.GetLinePort(lp)
				dist -= int64(p.Length - end.Precise)
			}
		}
	}
	return dist
}

// Traverse returns the Position when traversing from the port A of the first Line.
// If displacement is larger than the length of the path itself, ok = false.
// Displacement must not be negative.
// Note that this means the entire length of the first Line is traversed.
func (y *Layout) Traverse(path []LinePort, displacement int64) (pos Position, ok bool) {
	//log.Printf("Traverse(%#v, %d)", path, displacement)
	//defer func() {
	//	log.Printf("Traverse(%#v, %d) -> (%#v, %t)", path, displacement, pos, ok)
	//}()
	if displacement < 0 {
		panic("negative displacement")
	}
	prev := LinePort{
		LineI: path[0].LineI,
		PortI: PortA,
	}
	var current uint32
	for pathI := 0; pathI < len(path); pathI++ {
		cur := path[pathI]
		// prevConn is the LinePort on the same line as cur, and equivalent to prev (in terms of length from cur).
		var prevConn LinePort
		//log.Printf("prev %#v cur %#v count %d", prev, cur, current)
		// Either:
		//   a) prev points to a port that points to the same LineI as cur
		//   b) prev and cur both use the same LineI
		if cur.LineI != prev.LineI {
			// use ConnI, ConnP to make the LineIs equal
			_, p := y.GetLinePort(prev)
			prevConn = p.Conn()
			if cur.LineI != prevConn.LineI {
				panic("prev points to different line than cur")
			}
		} else {
			prevConn = prev
		}
		l := y.Lines[cur.LineI]
		// find length between prevConn and cur
		var length uint32
		if cur.PortI == -1 {
			// reached end of path
			return Position{}, false
		} else if cur.PortI == PortA {
			length = l.GetPort(prevConn.PortI).Length
		} else if prevConn.PortI == PortA {
			length = l.GetPort(cur.PortI).Length
		} else {
			panic("cannot go between B and C")
		}
		//log.Printf("length %d cl %d d %d", length, current+length, displacement)
		if current+length > uint32(displacement) {
			var pos Position
			delta := uint32(displacement) - current
			//log.Printf("delta %d", delta)
			switch cur.PortI {
			case PortA:
				switch prevConn.PortI {
				case PortA:
					panic("prevConn.PortI == cur.PortI")
				case PortB, PortC:
					_, p := y.GetLinePort(prevConn)
					pos = Position{cur.LineI, p.Length - delta, prevConn.PortI}
				}
			case PortB, PortC:
				pos = Position{cur.LineI, delta, cur.PortI}
			}
			if pos.Port == PortA {
				panic("invalid pos.Port")
			}
			return pos, true
		}
		current += length
		// add up length to current
		prev = cur
	}
	if current == uint32(displacement) {
		lp := path[len(path)-1]
		p := y.Lines[lp.LineI].GetPort(lp.PortI)
		var port PortI
		switch lp.PortI {
		case PortA:
			if len(path) != 1 {
				_, p := y.GetLinePort(path[len(path)-2])
				prevConn := p.Conn()
				switch prevConn.PortI {
				case PortA:
					// Assume the path "ends" here (as it points back to itself).
					return Position{}, false
					//panic("prevConn.PortI == cur.PortI")
				case PortB, PortC:
					port = prevConn.PortI
				}
			} else {
				// give up as we need FullPath for this
			}
		case PortB, PortC:
			port = lp.PortI
		}
		return Position{lp.LineI, p.Length, port}, true
	}
	// total length of the path was less than displacement
	return Position{}, false
}

// FullPath is a path including the start and intermediate points.
// The path does not overlap itself.
type FullPath struct {
	Start   LinePort
	Follows []LinePort
}

func (f FullPath) AtIndex(i int) LinePort {
	switch {
	case i < -1:
		panic("index less than -1")
	case i == -1:
		return f.Start
	default:
		return f.Follows[i]
	}
}

func (f FullPath) Clone() FullPath {
	follows := make([]LinePort, len(f.Follows))
	copy(follows, f.Follows)
	return FullPath{ // don't use keyed form to prevent missing fields
		f.Start,
		follows,
	}
}

func (a FullPath) Equal(b FullPath) bool {
	if a.Start != b.Start {
		return false
	}
	return slices.Equal(a.Follows, b.Follows)
}

func (y *Layout) MustFullPathTo(from, goal LinePort) FullPath {
	fp, err := y.FullPathTo(from, goal)
	if err != nil {
		panic(fmt.Sprintf("MustFullPathTo: %s", err))
	}
	return fp
}

type PathToSelfError struct{}

func (p PathToSelfError) Error() string {
	return "path to self (from == goal)"
}

func (y *Layout) FullPathTo(from, goal LinePort) (FullPath, error) {
	if from.PortI == PortDNC {
		from.PortI = PortA // choose an arbitrary port (it's a bad idea to have PortDNC as the Start, as then offset calculations cannot be made)
	}
	if from.LineI == goal.LineI {
		if from.PortI != PortDNC && from.PortI == goal.PortI {
			return FullPath{}, PathToSelfError{}
		}
		return FullPath{
			Start:   from,
			Follows: []LinePort{goal},
		}, nil
	}
	lps := y.PathTo(from.LineI, goal.LineI)
	start := lps[0]
	if from.PortI != start.PortI && from.PortI != PortA && from.PortI != PortDNC && start.PortI != PortA {
		return FullPath{}, fmt.Errorf("switchback necessary (from → start is %s → %s)", from, start)
	}
	_, p := y.GetLinePort(lps[len(lps)-1])
	end := p.Conn()
	if goal.PortI != end.PortI && goal.PortI != PortA && end.PortI != PortA {
		log.Printf("end %#v", end)
		return FullPath{}, fmt.Errorf("switchback necessary (goal → end is %s → %s)", goal, end)
	}
	lps = append(lps, goal)
	return FullPath{
		Start:   from,
		Follows: lps,
	}, nil
}

// PathTo returns a list of outgoing LinePorts in the order they should be followed.
// This assumes all switches can be operated in both normal and reverse directions.
func (y *Layout) PathTo(from, goal LineI) []LinePort {
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
		if LineI(i) == from {
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
		for pi := PortI(0); pi <= 2; pi++ {
			if debug {
				log.Printf("port %s", pi)
			}
			p := l.GetPort(pi)
			if !p.ConnFilled {
				if debug {
					log.Printf("unfilled")
				}
				continue
			}
			if pi+current.PortI == 5 {
				// cannot go between ports B and C
				if debug {
					log.Printf("between")
				}
				continue
			}
			if distance[p.ConnI] == infinity || distance[current.LineI] < distance[p.ConnI] {
				distance[p.ConnI] = distance[current.LineI] + 1
				using[p.ConnI] = LinePort{current.LineI, pi}
				queue = append(queue, LinePort{p.ConnI, pi})
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
	if debug {
		for i := range distance {
			log.Printf("distance[%d] = %d", i, distance[i])
		}
	}
	lps := make([]LinePort, distance[goal])
	for i, j := goal, 0; i != from; i, j = using[i].LineI, j+1 {
		lps[len(lps)-1-j] = using[i]
	}
	return lps
}

type Position struct {
	LineI LineI
	// Precise is the position from port A in µm.
	Precise uint32
	// Port is to where Precise is measuring to. This is needed to handle switches, where there could be two points where Precise is equal to 100000µm, on the normal and reverse sides of the switch.
	// For example, if Precise is measuring the position form port A to port C, this field would be PortC.
	Port PortI
}

func (y *Layout) ReversePath(path []LinePort) []LinePort {
	if path[len(path)-1].PortI == -1 {
		// The dummy LinePort is not needed here as it doesn't contain any more info.
		path = path[:len(path)-1]
	}
	res := make([]LinePort, len(path))
	for i := len(path) - 1; i >= 0; i-- {
		j := len(path) - 1 - i
		lp := path[i]
		_, p := y.GetLinePort(lp)
		res[j] = p.Conn()
	}
	return res
}

func (y *Layout) ReverseFullPath(fp FullPath) FullPath {
	res := make([]LinePort, len(fp.Follows)-1)
	for i := range res {
		j := len(fp.Follows) - 2 - i
		_, p := y.GetLinePort(fp.Follows[j])
		res[i] = p.Conn()
	}
	return FullPath{
		Start:   fp.Follows[len(fp.Follows)-1],
		Follows: append(res, fp.Start),
	}
}

func SameDirection(a, b FullPath) (same, ok bool) {
	// only consider Follows: Start is only useful with Follows[0]
	for _, alp := range a.Follows {
		for _, blp := range b.Follows {
			if alp.LineI == blp.LineI {
				if alp.PortI == PortA && blp.PortI != PortA {
					return false, true
				}
				if alp.PortI != PortA && blp.PortI == PortA {
					return false, true
				}
				if alp.PortI != PortA && blp.PortI != PortA {
					if alp.PortI == blp.PortI {
						return true, true
					} else {
						// two lines split here, but may converge again:
						//      *---*
						//     /     \
						// ---*-------*---
						// alp can take the top route, and blp can take the bottom route
						continue
					}
				}
				if alp.PortI == PortA && blp.PortI == PortA {
					return true, true
				}
			}
		}
	}
	return false, false
}

func (y *Layout) LinePortToPosition(lp LinePort) Position {
	_, p := y.GetLinePort(lp)
	return Position{
		LineI:   lp.LineI,
		Precise: p.Length,
		Port:    lp.PortI,
	}
}
