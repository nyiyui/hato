package layout

import (
	"errors"
	"fmt"

	"go.uber.org/zap"
	"golang.org/x/exp/slices"
)

type Offset = int64

// PositionToOffset2 returns the Offset of a Position on the same line as the given path.
func (y *Layout) PositionToOffset(fp FullPath, pos Position) (offset Offset) {
	offset, err := y.PositionToOffset2(fp, pos, PositionToOffsetOption{})
	if err != nil {
		panic(err)
	}
	return offset
}

// PositionToOffsetOption has options for PositionToOffset. The default is the zero value.
type PositionToOffsetOption struct {
	// DisallowPortMismatch makes PositionToOffset return PortMismatchError when the Position.Port does not match that of the matching LinePort of the Path.
	// Example:
	//   A=========B
	//        \
	//         --P-C
	//   path is taking the double-lined port/path
	//   P = Position
	// Here, DisallowPortMismatch would return an error as P is on a different port (same line) as the path.
	DisallowPortMismatch bool
}

// PortMismatchError is returned when the Position.Port does not match that of the corresponding LinePort of the Path. See PositionToOffsetOption for more details.
type PortMismatchError struct {
	fp  FullPath
	i   int
	pos Position
}

func (pm PortMismatchError) Error() string {
	return fmt.Sprintf("port mismatch: path has port %s, but position has port %s (path = %s ; current index = %d ; pos = %s)", pm.fp.Follows[pm.i].PortI, pm.pos.Port, pm.fp, pm.i, pm.pos)
}

// PositionToOffset2 returns the Offset of a Position on (optionally the same line as) the given path.
func (y *Layout) PositionToOffset2(fp FullPath, pos Position, option PositionToOffsetOption) (offset Offset, err error) {
	if pos.Precise != 0 && pos.Port == 0 {
		panic("invalid pos")
	}
	if pos.LineI == fp.Start.LineI {
		switch fp.Start.PortI {
		case PortA:
			if fp.Follows[0].PortI != pos.Port && pos.Precise != 0 {
				return 0, fmt.Errorf("pos not on path (different port on start) (pos = %s ; fp = %s)", pos, fp)
			}
			return int64(pos.Precise), nil
		case PortB, PortC:
			if fp.Follows[0].PortI != PortA {
				panic("follows[0] is B/C and start is B/C (cannot go between B and C)")
			}
			if fp.Start.PortI != pos.Port && !(pos.Port == PortA && pos.Precise == 0) {
				return 0, fmt.Errorf("pos not on path (different port on start) (pos = %s ; fp = %s)", pos, fp)
			}
			_, p := y.GetLinePort(fp.Start)
			return int64(p.Length) - int64(pos.Precise), nil
		default:
			panic(fmt.Sprintf("invalid port %s (path: %#v)", fp.Start.PortI, fp))
		}
	}
	var cum int64
	if i := slices.IndexFunc(fp.Follows, func(lp LinePort) bool { return lp.LineI == pos.LineI }); i == -1 {
		return 0, errors.New("not on follows")
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
			if lp.PortI != pos.Port && option.DisallowPortMismatch {
				return 0, PortMismatchError{
					fp:  fp,
					i:   i,
					pos: pos,
				}
			}
			// last
			return cum + y.positionToStep(prev, lp, pos, fp), nil
		} else {
			cum += y.distanceBetween(prev, lp)
		}
	}
	panic("unreachable67")
}

func (y *Layout) positionToStep(a, b LinePort, pos Position, fullPathForDebug FullPath) int64 {
	//log.Printf("a %#v", a)
	//log.Printf("b %#v", b)
	//log.Printf("pos %d", pos)
	origA := a
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
			panic("unreachable85")
		case PortB, PortC:
			// A → B/C
			return int64(pos.Precise)
		default:
			panic("unreachable90")
		}
	case PortB, PortC:
		if b.PortI != PortA {
			panic(fmt.Sprintf("positionToStep a = %s or %s, b = %s: B/C to B/C, path = %s", a, origA, b, fullPathForDebug))
		}
		// B/C → A
		_, p := y.GetLinePort(a)
		return int64(p.Length) - int64(pos.Precise)
	default:
		panic("unreachable100")
	}
}

// OffsetToPosition returns a Position from an Offset, starting at the start of the FullPath.
func (y *Layout) MustOffsetToPosition(fp FullPath, offset Offset) Position {
	pos, err := y.OffsetToPosition(fp, offset)
	if err != nil {
		panic(err)
	}
	return pos
}

// OffsetToPosition returns a Position from an Offset, starting at the start of the FullPath.
func (y *Layout) OffsetToPosition(fp FullPath, offset Offset) (pos Position, err error) {
	zap.S().Debugf("OffsetToPosition(%s, %d)", fp, offset)
	if offset < 0 {
		return Position{}, errors.New("negative offset")
	}
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
		zap.S().Debugf("step %d, nextCum %d", step, nextCum)
		if nextCum > offset {
			move := offset - cum
			//log.Printf("i %d", i)
			// hmm why was distanceBetween > 0 if prev == cur
			// hmmmmmmmmmm move and offset are both negative hmmmm
			return y.stepToPosition(prev, cur, move, fp), nil
			// sometimes (when offset is small enough?) prev (FullPath.Start) == cur (first in FullPath.Follows)
			// ↑ can cause a bug with unreachable194
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
				panic(fmt.Sprintf("unreachable (a.PortI==b.PortI==PortA) (a=%s b=%s fp=%s)", a, b, fp))
			case PortB, PortC:
				// A → B/C
				_, p := y.GetLinePort(b)
				return Position{
					LineI:   a.LineI,
					Precise: p.Length,
					Port:    b.PortI,
				}, nil
			default:
				panic("unreachable (invalid port)")
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
			}, nil
		default:
			panic("unreachable (invalid port)")
		}
	}
	return Position{}, fmt.Errorf("offset overran path (cum=%d)", cum)
}

func (y *Layout) stepToPosition(a, b LinePort, move int64, fullPathForDebug FullPath) Position {
	//log.Printf("a %#v", a)
	//log.Printf("b %#v", b)
	//log.Printf("move %d", move)
	aOrig := a
	if a.LineI != b.LineI {
		_, p := y.GetLinePort(a)
		a = p.Conn()
		if a.LineI != b.LineI {
			panic(fmt.Sprintf("LinePort A not connected to same line as LinePort B %s", fullPathForDebug))
		}
	}
	switch a.PortI {
	case PortA:
		switch b.PortI {
		case PortA:
			panic(fmt.Sprintf("unreachable194 a %s or %s b %s move %d %s",
				a, aOrig, b, move, fullPathForDebug,
			)) // TODO: happens a lot
		case PortB, PortC:
			// A → B/C
			return Position{
				LineI:   a.LineI,
				Precise: uint32(move),
				Port:    b.PortI,
			}
		default:
			panic("unreachable203")
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
		panic("unreachable217")
	}
}

// distanceBetween calculates the distance between two LinePorts A and B.
// LinePort A must connect to a LinePort with the same line as LinePort B.
func (y *Layout) distanceBetween(a, b LinePort) int64 {
	if a.LineI != b.LineI {
		_, p := y.GetLinePort(a)
		a = p.Conn()
		if a.LineI != b.LineI {
			panic(fmt.Sprintf("LinePort A not connected to same line as LinePort B (a = %s ; b = %s)", a, b))
		}
	}
	if a == b {
		return 0
	}
	switch a.PortI {
	case PortA:
		switch b.PortI {
		case PortA:
			panic("unreachable238")
		case PortB, PortC:
			_, p := y.GetLinePort(b)
			return int64(p.Length)
		default:
			panic("unreachable243")
		}
	case PortB, PortC:
		if b.PortI != PortA {
			panic(fmt.Sprintf("B/C to B/C: %#v %#v", a, b))
		}
		_, p := y.GetLinePort(a)
		return int64(p.Length)
	default:
		panic("unreachable252")
	}
}
