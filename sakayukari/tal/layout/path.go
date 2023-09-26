package layout

import (
	"errors"
	"fmt"

	"golang.org/x/exp/slices"
)

type Offset = int64

// PositionToOffset returns an Offset where the first LinePort of path is the starting point.
func (y *Layout) PositionToOffset(fp FullPath, pos Position) (offset Offset) {
	if pos.Precise != 0 && pos.Port == 0 {
		panic("invalid pos")
	}
	if pos.LineI == fp.Start.LineI {
		switch fp.Start.PortI {
		case PortA:
			if fp.Follows[0].PortI != pos.Port {
				panic("pos not on path (different port on start)")
			}
			return int64(pos.Precise)
		case PortB, PortC:
			if fp.Follows[0].PortI != PortA {
				panic("follows[0] is B/C and start is B/C (cannot go between B and C)")
			}
			if fp.Start.PortI != pos.Port {
				panic("pos not on path (different port on start)")
			}
			_, p := y.GetLinePort(fp.Start)
			return int64(p.Length) - int64(pos.Precise)
		default:
			panic(fmt.Sprintf("invalid port %s (path: %#v)", fp.Start.PortI, fp))
		}
	}
	var cum int64
	if i := slices.IndexFunc(fp.Follows, func(lp LinePort) bool { return lp.LineI == pos.LineI }); i == -1 {
		panic("pos not on follows")
	}
	for i, lp := range fp.Follows {
		var prev LinePort
		if i == 0 {
			prev = fp.Start
		} else {
			prev = fp.Follows[i-1]
		}
		if lp.LineI == pos.LineI && lp.PortI != PortA && lp.PortI != pos.Port {
			panic(fmt.Sprintf("pos %#v strays from path %#v on index %d", pos, fp.Follows, i))
		}
		if lp.LineI == pos.LineI {
			// last
			//panic("TODO")
			return cum + y.positionToStep(prev, lp, pos)
		} else {
			cum += y.distanceBetween(prev, lp)
		}
	}
	panic("unreachable")
}

func (y *Layout) positionToStep(a, b LinePort, pos Position) int64 {
	//log.Printf("a %#v", a)
	//log.Printf("b %#v", b)
	//log.Printf("pos %d", pos)
	if a.LineI != b.LineI {
		_, p := y.GetLinePort(a)
		a = p.Conn()
		if a.LineI != b.LineI {
			panic("LinePort A not connected to same line as LinePort B")
		}
	}
	switch a.PortI {
	case PortA:
		switch b.PortI {
		case PortA:
			panic("unreachable")
		case PortB, PortC:
			// A → B/C
			return int64(pos.Precise)
		default:
			panic("unreachable")
		}
	case PortB, PortC:
		if b.PortI != PortA {
			panic("B/C to B/C")
		}
		// B/C → A
		_, p := y.GetLinePort(a)
		return int64(p.Length) - int64(pos.Precise)
	default:
		panic("unreachable")
	}
}

// PositionToOffset returns a Position from an Offset, starting at the start of the FullPath.
func (y *Layout) OffsetToPosition(fp FullPath, offset Offset) (pos Position) {
	var cum int64
	for i, cur := range fp.Follows {
		var prev LinePort
		if i == 0 {
			prev = fp.Start
		} else {
			prev = fp.Follows[i-1]
		}
		step := y.distanceBetween(prev, cur)
		nextCum := cum + step
		if nextCum > offset {
			move := offset - cum
			//log.Printf("i %d", i)
			return y.stepToPosition(prev, cur, move)
		}
		cum += step
	}
	if cum == offset {
		a := fp.Follows[len(fp.Follows)-2]
		b := fp.Follows[len(fp.Follows)-1]
		if a.LineI != b.LineI {
			_, p := y.GetLinePort(a)
			a = p.Conn()
			if a.LineI != b.LineI {
				panic("LinePort A not connected to same line as LinePort B")
			}
		}
		// for debugging
		// log.Printf("fp %#v", fp)
		// log.Printf("a %#v", a)
		// log.Printf("b %#v", b)
		switch a.PortI {
		case PortA:
			switch b.PortI {
			case PortA:
				panic("unreachable")
			case PortB, PortC:
				// A → B/C
				_, p := y.GetLinePort(b)
				return Position{
					LineI:   a.LineI,
					Precise: p.Length,
					Port:    b.PortI,
				}
			default:
				panic("unreachable")
			}
		case PortB, PortC:
			if b.PortI != PortA {
				panic("B/C to B/C")
			}
			// B/C → A
			return Position{
				LineI:   a.LineI,
				Precise: 0,
				Port:    a.PortI,
			}
		default:
			panic("unreachable")
		}
	}
	panic(fmt.Sprintf("offset overran path (cum=%d)", cum))
}

func (y *Layout) stepToPosition(a, b LinePort, move int64) Position {
	//log.Printf("a %#v", a)
	//log.Printf("b %#v", b)
	//log.Printf("move %d", move)
	if a.LineI != b.LineI {
		_, p := y.GetLinePort(a)
		a = p.Conn()
		if a.LineI != b.LineI {
			panic("LinePort A not connected to same line as LinePort B")
		}
	}
	switch a.PortI {
	case PortA:
		switch b.PortI {
		case PortA:
			panic("unreachable")
		case PortB, PortC:
			// A → B/C
			return Position{
				LineI:   a.LineI,
				Precise: uint32(move),
				Port:    b.PortI,
			}
		default:
			panic("unreachable")
		}
	case PortB, PortC:
		if b.PortI != PortA {
			panic("B/C to B/C")
		}
		// B/C → A
		_, p := y.GetLinePort(a)
		return Position{
			LineI:   a.LineI,
			Precise: uint32(int64(p.Length) - move),
			Port:    a.PortI,
		}
	default:
		panic("unreachable")
	}
}

// distanceBetween calculates the distance between two LinePorts A and B.
// LinePort A must connect to a LinePort with the same line as LinePort B.
func (y *Layout) distanceBetween(a, b LinePort) int64 {
	if a.LineI != b.LineI {
		_, p := y.GetLinePort(a)
		a = p.Conn()
		if a.LineI != b.LineI {
			panic("LinePort A not connected to same line as LinePort B")
		}
	}
	if a == b {
		return 0
	}
	switch a.PortI {
	case PortA:
		switch b.PortI {
		case PortA:
			panic("unreachable")
		case PortB, PortC:
			_, p := y.GetLinePort(b)
			return int64(p.Length)
		default:
			panic("unreachable")
		}
	case PortB, PortC:
		if b.PortI != PortA {
			panic(fmt.Sprintf("B/C to B/C: %#v %#v", a, b))
		}
		_, p := y.GetLinePort(a)
		return int64(p.Length)
	default:
		panic("unreachable")
	}
}

func SameDirection2(prev, next []LinePort) (sameDir bool, err error) {
	if prev[0] == next[0] {
		return true, nil
	}
	if prev[len(prev)-1].LineI == next[0].LineI {
		return false, nil
	}
	return false, errors.New("idk")
}

func SameDir1(a, b LinePort) bool {
	if a.LineI != b.LineI {
		panic("different Line")
	}
	aSame := a.PortI == PortA
	bSame := b.PortI == PortA
	return aSame == bSame
}
