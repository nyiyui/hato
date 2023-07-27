package layout

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"nyiyui.ca/hato/sakayukari/conn"
)

func TestLineID(t *testing.T) {
	li := LineID{
		Conn: conn.Id{
			Type:     "t",
			Variant:  "v",
			Instance: "i",
		},
		Line: "l",
	}
	var li2 LineID
	data, err := json.Marshal(li)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s", data)
	err = json.Unmarshal(data, &li2)
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(li2, li) {
		t.Fatal(cmp.Diff(li2, li))
	}
}

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
	countLength := func(y Layout, connI LineI, portI PortI, destI LineI) uint32 {
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
	y, err := InitTestbench3()
	if err != nil {
		t.Fatalf("error: %s", err)
	}
	data, err := json.MarshalIndent(y, "", "  ")
	if err != nil {
		t.Fatalf("json: %s", err)
	}
	t.Logf("%s", data)
}

/*
func TestReversePath(t *testing.T) {
	y, err := InitTestbench3()
	if err != nil {
		t.Fatalf("InitTestbench3: %s", err)
	}
	type setup struct {
		y        *Layout
		path     []LinePort
		reversed []LinePort
	}
	setups := []setup{
		{y, y.PathTo(y.MustLookupIndex("Z"), y.MustLookupIndex("W")), y.PathTo(y.MustLookupIndex("W"), y.MustLookupIndex("Z"))},
	}
	for i, s := range setups {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			got := y.reversePath(s.path)
			want := s.reversed
			t.Logf("path: %#v", s.path)
			t.Logf("got:  %#v", got)
			t.Logf("want: %#v", want)
			if !cmp.Equal(got, want) {
				t.Fatalf("mismatch: %s", cmp.Diff(got, want))
			}
		})
	}
}
*/

func TestTraverse(t *testing.T) {
	y, err := InitTestbench3()
	if err != nil {
		t.Fatalf("InitTestbench3: %s", err)
	}
	path := y.PathTo(y.MustLookupIndex("Z"), y.MustLookupIndex("W"))
	path2 := y.PathTo(y.MustLookupIndex("W"), y.MustLookupIndex("Z"))
	type setup struct {
		path         []LinePort
		displacement int64
		final        Position
	}
	var base uint32
	base += y.Lines[y.MustLookupIndex("Z")].PortB.Length
	base += y.Lines[y.MustLookupIndex("Y")].PortB.Length
	t.Logf("path: %#v", path)
	t.Logf("path2: %#v", path2)
	setups := []setup{
		{path, 0, Position{y.MustLookupIndex("Z"), 0, PortB}},
		{path, 123456, Position{y.MustLookupIndex("Z"), 123456, PortB}},
		{path, 256000, Position{y.MustLookupIndex("Y"), 128000, PortB}},
		{path, int64(base), Position{y.MustLookupIndex("X"), 0, PortB}},
		{path, int64(base) + 1, Position{y.MustLookupIndex("X"), 1, PortB}},
		{path, 128000 + 872000, Position{y.MustLookupIndex("X"), 0, PortB}},
		{path2, 0, Position{y.MustLookupIndex("X"), y.Lines[y.MustLookupIndex("X")].PortB.Length, PortB}},
		{path2, 1, Position{y.MustLookupIndex("X"), y.Lines[y.MustLookupIndex("X")].PortB.Length - 1, PortB}},
		{path2, 123, Position{y.MustLookupIndex("X"), y.Lines[y.MustLookupIndex("X")].PortB.Length - 123, PortB}},
		{path2, 628964 + 872000, Position{y.MustLookupIndex("Y"), 0, PortB}},
	}
	// TODO: negative traversal testing
	for i, setup := range setups {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			pos, ok := y.Traverse(setup.path, setup.displacement)
			if !ok {
				t.Fatalf("!ok")
			}
			if !cmp.Equal(pos, setup.final) {
				t.Fatalf("Position mismatch:\n%s", cmp.Diff(pos, setup.final))
			}
		})
	}
}

func TestCount(t *testing.T) {
	y, err := InitTestbench3()
	if err != nil {
		t.Fatalf("InitTestbench3: %s", err)
	}
	Z := y.MustLookupIndex("Z")
	Y := y.MustLookupIndex("Y")
	X := y.MustLookupIndex("X")
	W := y.MustLookupIndex("W")
	V := y.MustLookupIndex("V")
	_, _, _, _, _ = Z, Y, X, W, V
	path := y.PathTo(y.MustLookupIndex("Z"), y.MustLookupIndex("W"))
	path2 := y.PathTo(y.MustLookupIndex("W"), y.MustLookupIndex("Z"))
	type setup struct {
		y     *Layout
		path  []LinePort
		start Position
		end   Position
		want  int64
	}
	var base uint32
	base += y.Lines[y.MustLookupIndex("Z")].PortB.Length
	base += y.Lines[y.MustLookupIndex("Y")].PortB.Length
	t.Logf("path: %#v", path)
	t.Logf("path2: %#v", path2)
	setups := []setup{
		{y, y.PathTo(Z, X), Position{Z, 10, 0}, Position{Y, 0, 0}, int64(y.Lines[Z].PortB.Length - 10)},
		// TODO: more tests
	}
	// TODO: negative traversal testing
	for i, setup := range setups {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Logf("path: %#v", setup.path)
			t.Logf("start: %#v", setup.start)
			t.Logf("end: %#v", setup.end)
			dist := y.Count(setup.path, setup.start, setup.end)
			if !cmp.Equal(dist, setup.want) {
				t.Fatalf("want %d, got %d", setup.want, dist)
			}
		})
	}
}
func TestReversePath(t *testing.T) {
	y, err := InitTestbench3()
	if err != nil {
		t.Fatalf("InitTestbench3: %s", err)
	}
	Z := y.MustLookupIndex("Z")
	Y := y.MustLookupIndex("Y")
	X := y.MustLookupIndex("X")
	W := y.MustLookupIndex("W")
	V := y.MustLookupIndex("V")
	_, _, _, _, _ = Z, Y, X, W, V
	path := y.PathTo(y.MustLookupIndex("Z"), y.MustLookupIndex("W"))
	path2 := y.PathTo(y.MustLookupIndex("W"), y.MustLookupIndex("Z"))
	type setup struct {
		y    *Layout
		path []LinePort
		want []LinePort
	}
	var base uint32
	base += y.Lines[y.MustLookupIndex("Z")].PortB.Length
	base += y.Lines[y.MustLookupIndex("Y")].PortB.Length
	t.Logf("path: %#v", path)
	t.Logf("path2: %#v", path2)
	setups := []setup{
		{y, y.PathTo(Z, X), y.PathTo(X, Z)},
		{y, y.PathTo(Z, W), y.PathTo(W, Z)},
		{y, y.PathTo(Z, V), y.PathTo(V, Z)},
		{y, y.PathTo(V, Z), y.PathTo(Z, V)},
		// TODO: more tests
	}
	// TODO: negative traversal testing
	for i, setup := range setups {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Logf("setup: %#v", setup)
			res := y.ReversePath(setup.path)
			if !cmp.Equal(res, setup.want) {
				t.Fatalf("want %d, got %d", setup.want, res)
			}
		})
	}
}
