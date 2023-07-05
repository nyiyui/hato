package layout

import "testing"

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

func TestTestbench(t *testing.T) {
	_, err := InitTestbench()
	if err != nil {
		t.Fatalf("error: %s", err)
	}
}
