package layout

import "testing"

func TestTestbench6(t *testing.T) {
	y, err := InitTestbench6()
	if err != nil {
		t.Fatal(err)
	}
	y.PathTo(y.MustLookupIndex("snb4"), y.MustLookupIndex("mitouc2"))
	y.PathTo(y.MustLookupIndex("nagase1"), y.MustLookupIndex("mitouc2"))
	y.PathTo(y.MustLookupIndex("nagase1"), y.MustLookupIndex("mitouc3"))
	y.PathTo(y.MustLookupIndex("nagase1"), y.MustLookupIndex("snb4"))
}
