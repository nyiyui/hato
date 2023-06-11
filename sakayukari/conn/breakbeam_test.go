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
