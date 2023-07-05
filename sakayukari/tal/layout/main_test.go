package layout

import (
	"testing"

	"github.com/google/go-cmp/cmp"
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

func TestPathTo(t *testing.T) {
	y, err := Connect([]Line{
		StraightLine(312000),
		StraightLine(312000),
		StraightLine(312000),
	})
	if err != nil {
		t.Fatalf("error: %s", err)
	}
	y.Lines[1].PortA.ConnI = 2
	y.Lines[1].PortB.ConnI = 0
	t.Logf("layout: %#v", y)
	expected := []LinePort{LinePort{LineI: 0, PortI: 1}, LinePort{LineI: 1, PortI: 0}}
	got := y.PathTo(0, 2)
	if !cmp.Equal(got, expected) {
		t.Logf("PathTo expected: %#v", expected)
		t.Logf("PathTo got: %#v", got)
		t.Fatalf("PathTo diff: %s", cmp.Diff(got, expected))
	}
}

func TestTestbench(t *testing.T) {
	_, err := InitTestbench()
	if err != nil {
		t.Fatalf("error: %s", err)
	}
}
