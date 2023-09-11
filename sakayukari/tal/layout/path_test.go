package layout

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestOffsets(t *testing.T) {
	y, err := InitTestbench3()
	if err != nil {
		t.Fatalf("InitTestbench3: %s", err)
	}
	Z := y.MustLookupIndex("Z")
	ZA := LinePort{Z, PortA}
	ZB := LinePort{Z, PortB}
	//Y := y.MustLookupIndex("Y")
	//YA := LinePort{Y, PortA}
	//YB := LinePort{Y, PortB}
	//X := y.MustLookupIndex("X")
	//XA := LinePort{X, PortA}
	//XB := LinePort{X, PortB}
	//XC := LinePort{X, PortC}
	W := y.MustLookupIndex("W")
	WA := LinePort{W, PortA}
	WB := LinePort{W, PortB}
	V := y.MustLookupIndex("V")
	VA := LinePort{V, PortA}
	VB := LinePort{V, PortB}
	type setup struct {
		name      string
		y         *Layout
		path      FullPath
		offset    int64
		offsetSet bool
		pos       Position
		posSet    bool
	}
	setups := []setup{}
	for _, p := range []int64{0, 1, 128000, 129000, 257000, 385000} {
		setups = append(setups, []setup{
			{fmt.Sprintf("ZA→VB-%d", p), y, y.MustFullPathTo(ZA, VB), p, true, Position{}, false},
			{fmt.Sprintf("ZA→WB-%d", p), y, y.MustFullPathTo(ZA, WB), p, true, Position{}, false},
		}...)
		if p <= 128000*3 {
			setups = append(setups, []setup{
				{fmt.Sprintf("VA→ZA-%d", p), y, y.MustFullPathTo(VA, ZA), p, true, Position{}, false},
				{fmt.Sprintf("WA→ZA-%d", p), y, y.MustFullPathTo(WA, ZA), p, true, Position{}, false},
				{fmt.Sprintf("ZA→VA-%d", p), y, y.MustFullPathTo(ZA, VA), p, true, Position{}, false},
				{fmt.Sprintf("ZA→WA-%d", p), y, y.MustFullPathTo(ZA, WA), p, true, Position{}, false},
			}...)
		}
		if p <= 256000 {
			setups = append(setups, []setup{
				{fmt.Sprintf("VA→ZB-%d", p), y, y.MustFullPathTo(VA, ZB), p, true, Position{}, false},
				{fmt.Sprintf("WA→ZB-%d", p), y, y.MustFullPathTo(WA, ZB), p, true, Position{}, false},
			}...)
		}
	}
	for i, s := range setups {
		t.Run(fmt.Sprintf("%d-%s", i, s.name), func(t *testing.T) {
			if !s.offsetSet && !s.posSet {
				panic("either offset or pos must be set")
			}
			if !s.posSet {
				s.pos = s.y.OffsetToPosition(s.path, s.offset)
			}
			if !s.offsetSet {
				s.offset = s.y.PositionToOffset(s.path, s.pos)
			}
			pos2 := s.y.OffsetToPosition(s.path, s.offset)
			if !cmp.Equal(s.pos, pos2) {
				t.Logf("s.pos %#v", s.pos)
				t.Logf("s.offset %#v", s.offset)
				t.Logf("pos2 %#v", pos2)
				t.Fatalf("OffsetToPosition failed")
			}
			t.Logf("s.offset %#v", s.offset)
			t.Logf("s.pos %#v", s.pos)
			offset2 := s.y.PositionToOffset(s.path, s.pos)
			if !cmp.Equal(s.offset, offset2) {
				t.Logf("s.offset %#v", s.offset)
				t.Logf("s.pos %#v", s.pos)
				t.Logf("offset2 %#v", offset2)
				t.Fatalf("PositionToOffset failed")
			}
		})
	}
}
