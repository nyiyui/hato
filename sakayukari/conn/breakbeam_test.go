package conn

import (
	"fmt"
	"reflect"
	"testing"
)

type testCase struct {
	Line      string
	Values    map[string]bool
	Monotonic int64
}

func TestParse(t *testing.T) {
	cases := []testCase{
		{"A1B1C1T838942", map[string]bool{"A": true, "B": true, "C": true}, 838942},
	}
	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			values, monotonic, err := parse(tc.Line)
			if err != nil {
				t.Fatal(err)
			}
			if monotonic != tc.Monotonic {
				t.Fatal("monotonic mismatch")
			}
			if !reflect.DeepEqual(values, tc.Values) {
				t.Fatal("values mismatch")
			}
		})
	}
}

func TestHistory(t *testing.T) {
	pointA, pointB := 0, 1
	s := &velocity2State{
		History: []velocity2Single{
			{2, []bool{true, true, true}},
			{1, []bool{true, false, true}},
			{0, []bool{true, false, true}},
		},
	}
	if s.GetHistory(pointA, pointB, 0).Monotonic != 2 {
		t.Fatal("GetHistory 0 failed")
	}
	if s.GetHistory(pointA, pointB, 1).Monotonic != 0 {
		t.Fatal("GetHistory 1 failed")
	}
}
