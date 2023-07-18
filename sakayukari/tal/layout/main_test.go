package layout

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestConnect(t *testing.T) {
	y, err := Connect([]Line{
		StraightLine(312000),
		StraightLine(312000),
	})
	var expected uint32 = 312000 + 312000
	data, _ := json.MarshalIndent(y, "", "  ")
	t.Logf("layout: %s", data)
	if err != nil {
		t.Fatalf("error: %s", err)
	}
	countLength := func(y Layout, connI LineI, portI int, destI LineI) uint32 {
		i := connI
		p := portI
		var sum uint32 = 0
		for {
			//t.Logf("i %d p %d sum %d", i, p, sum)
			l := y.Lines[i]
			port := l.GetPort(p)
			sum += port.Length
			if i != destI {
				if !port.ConnFilled {
					t.Fatalf("encountered unfilled port on line %d port %d", i, p)
				}
				i = port.ConnI
				if port.ConnP == 0 {
					p = 1
				} else if port.ConnP == 1 {
					p = 0
				} else {
					panic("unexpected port.ConnP")
				}
			} else {
				break
			}
		}
		return sum
	}

	if got := countLength(y, 0, 1, 1); got != expected {
		t.Fatalf("expected %d, got %d", expected, got)
	}
}

func MustConnect(t *testing.T, lines []Line) *Layout {
	y, err := Connect(lines)
	if err != nil {
		t.Fatalf("connect: %s", err)
	}
	return &y
}

func TestPathTo(t *testing.T) {
	type setup struct {
		Comment string
		Layout  *Layout
		From    LineI
		Goal    LineI
	}
	y := MustConnect(t, []Line{
		StraightLine(312000),
		StraightLine(312000),
		StraightLine(312000),
	})
	y.Lines[1].PortA.ConnI = 2
	y.Lines[1].PortB.ConnI = 0
	testbench2, err := InitTestbench2()
	if err != nil {
		t.Fatalf("InitTestbench2: %s", err)
	}
	testbench3, err := InitTestbench3()
	if err != nil {
		t.Fatalf("InitTestbench2: %s", err)
	}
	setups := []setup{
		{"straight-ascend", y, 0, 2},
		{"straight-descend", y, 2, 0},
		{"testbench2-normal", testbench2, 0, 1},
		{"testbench2-reverse", testbench2, 0, 2},
		{"testbench2-YW", testbench2, testbench2.MustLookupIndex("Y"), testbench2.MustLookupIndex("W")},
		{"testbench3-ZW", testbench3, testbench3.MustLookupIndex("Z"), testbench3.MustLookupIndex("Y")},
	}
	for i, s := range setups {
		t.Run(fmt.Sprintf("%d-%s", i, s.Comment), func(t *testing.T) {
			y := s.Layout
			data, _ := json.MarshalIndent(y, "", "  ")
			t.Logf("layout-json: %s", data)
			path := y.PathTo(s.From, s.Goal)
			var current LineI = -1
			for i, lp := range path {
				next := y.Lines[lp.LineI].GetPort(lp.PortI)
				t.Logf("%d: %d → %s", i, current, next)
				current = next.ConnI
			}
			if current != s.Goal {
				t.Fatalf("did not reach goal (%d): reached %d instead", s.Goal, current)
			}
		})
	}
}

func TestTestbench1(t *testing.T) {
	_, err := InitTestbench1()
	if err != nil {
		t.Fatalf("error: %s", err)
	}
}

func TestTestbench2(t *testing.T) {
	_, err := InitTestbench2()
	if err != nil {
		t.Fatalf("error: %s", err)
	}
}

func TestTestbench3(t *testing.T) {
	_, err := InitTestbench3()
	if err != nil {
		t.Fatalf("error: %s", err)
	}
}
