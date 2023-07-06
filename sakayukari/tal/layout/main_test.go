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
	t.Logf("layout: %#v", y)
	if err != nil {
		t.Fatalf("error: %s", err)
	}
	if got := y.countLength(); got != expected {
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
		From    int
		Goal    int
	}
	y := MustConnect(t, []Line{
		StraightLine(312000),
		StraightLine(312000),
		StraightLine(312000),
	})
	y.Lines[1].PortA.ConnI = 2
	y.Lines[1].PortB.ConnI = 0
	setups := []setup{
		{"normal", y, 0, 2},
		{"reverse", y, 2, 0},
	}
	for i, s := range setups {
		t.Run(fmt.Sprintf("%d-%s", i, s.Comment), func(t *testing.T) {
			t.Logf("layout: %#v", y)
			data, _ := json.Marshal(y)
			t.Logf("layout-json: %s", data)
			path := y.PathTo(s.From, s.Goal)
			current := -1
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

func TestTestbench(t *testing.T) {
	_, err := InitTestbench()
	if err != nil {
		t.Fatalf("error: %s", err)
	}
}
