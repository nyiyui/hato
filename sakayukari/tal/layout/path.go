package layout

import (
	"errors"
	"fmt"

	"golang.org/x/exp/slices"
)

type Offset = int64

// PositionToOffset returns an Offset where the first LinePort of path is the starting point.
func (y *Layout) PositionToOffset(fp FullPath, pos Position) (offset Offset) {
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
		}
		panic("unreachable")
	}
	var cum int64
	switch fp.Start.PortI {
	case PortA:
		_, p := y.GetLinePort(fp.Follows[0])
		cum += int64(p.Length)
	case PortB, PortC:
		_, p := y.GetLinePort(fp.Start)
		cum += int64(p.Length)
	}
	prev := fp.Start
	if i := slices.IndexFunc(fp.Follows, func(lp LinePort) bool { return lp.LineI == pos.LineI }); i == -1 {
		panic("pos not on follows")
	}
	for i, lp := range fp.Follows {
		if lp.PortI != pos.Port {
			panic(fmt.Sprintf("pos strays from path on index %d", i))
		}
		if lp.LineI == pos.LineI {
			// last
			switch lp.PortI {
			case PortA:
				cum += int64(pos.Precise)
			case PortB, PortC:
				_, p := y.GetLinePort(lp)
				cum += int64(p.Length) - int64(pos.Precise)
			}
			return cum
		} else {
			cum += y.distanceBetween(prev, lp)
		}
		prev = lp
	}
	panic("unreachable")
}

// PositionToOffset returns a Position from an Offset.
func (y *Layout) OffsetToPosition(fp FullPath, offset Offset) (pos Position) {
	panic("not implemented yet")
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
			panic("B/C to B/C")
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
