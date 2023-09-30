package layout

import (
	"fmt"

	"golang.org/x/exp/slices"
)

// TODO: support joining by just first/last, instead of first/last and port.

type JoinI int

const (
	JoinIFirst = JoinI(1)
	JoinILast  = JoinI(2)
)

func (j JoinI) String() string {
	switch j {
	case JoinIFirst:
		return "F"
	case JoinILast:
		return "L"
	default:
		return fmt.Sprint(int(j))
	}
}

type Segment struct {
	Prefix string
	Lines  []Line
	First  *SegmentJoin
	Last   *SegmentJoin
}

type SegmentJoin struct {
	MyPort       PortI
	TargetPrefix string
	TargetJoin   JoinI
	TargetPort   PortI
}

func Connect2(segments []Segment) (Layout, error) {
	// append lines of all segments together
	layouts := make([]Layout, len(segments))
	totalLen := 0
	for i, segment := range segments {
		var err error
		layouts[i], err = Connect(segment.Lines)
		if err != nil {
			return Layout{}, fmt.Errorf("segment %d (%s): %w", i, segment.Prefix, err)
		}
		totalLen += len(layouts[i].Lines)
	}
	type place struct {
		First LineI
		Last  LineI
	}
	combined := Layout{Lines: make([]Line, 0, totalLen)}
	places := make([]place, 0, len(layouts))
	for _, layout := range layouts {
		places = append(places, place{
			First: LineI(len(combined.Lines)),
			Last:  LineI(len(combined.Lines) + len(layout.Lines)),
		})
		combined.Lines = append(combined.Lines, layout.Lines...)
	}

	// join segments in combined layout
	for i, segment := range segments {
		place := places[i]
		if join := segment.First; join != nil {
			targetI := slices.IndexFunc(segments, func(s Segment) bool { return s.Prefix == join.TargetPrefix })
			if targetI == -1 {
				return Layout{}, fmt.Errorf("joining segment %d (%s): ConnPrefix (%s) for target of first join not found", i, segment.Prefix, join.TargetPrefix)
			}
			joinPlace := places[targetI]
			p := combined.Lines[place.First].GetPort(join.TargetPort)
			switch join.TargetJoin {
			case JoinIFirst:
				p.ConnI = joinPlace.First
			case JoinILast:
				p.ConnI = joinPlace.Last
			default:
				panic("unreachable")
			}
			p.ConnP = join.TargetPort
			combined.Lines[place.First].SetPort(join.TargetPort, p)
		}
		if join := segment.Last; join != nil {
			targetI := slices.IndexFunc(segments, func(s Segment) bool { return s.Prefix == join.TargetPrefix })
			if targetI == -1 {
				return Layout{}, fmt.Errorf("joining segment %d (%s): ConnPrefix (%s) for target of first join not found", i, segment.Prefix, join.TargetPrefix)
			}
			joinPlace := places[targetI]
			p := combined.Lines[place.Last].GetPort(join.TargetPort)
			switch join.TargetJoin {
			case JoinIFirst:
				p.ConnI = joinPlace.First
			case JoinILast:
				p.ConnI = joinPlace.Last
			default:
				panic("unreachable")
			}
			p.ConnP = join.TargetPort
			combined.Lines[place.Last].SetPort(join.TargetPort, p)
		}
	}
	return combined, nil
}
