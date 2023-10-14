package tal

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"nyiyui.ca/hato/sakayukari/tal/layout"
)

func TestMinimumConstraint(t *testing.T) {
	y, err := layout.InitTestbench6()
	if err != nil {
		t.Fatal(err)
	}
	path := layout.FullPath{
		Start: layout.LinePort{
			LineI: y.MustLookupIndex("nagase1"),
			PortI: layout.PortA,
		},
		Follows: []layout.LinePort{
			{
				LineI: y.MustLookupIndex("nagase1"),
				PortI: layout.PortB,
			},
			{
				LineI: y.MustLookupIndex("mitouc2"),
				PortI: layout.PortB,
			},
			{
				LineI: y.MustLookupIndex("snb4"),
				PortI: layout.PortA,
			},
		},
	}
	{
		ca := Constraint{
			Path:        path,
			After:       y.MustOffsetToPosition(path, 0),
			AfterFilled: true,
			Before: layout.Position{
				LineI:   y.MustLookupIndex("nagase1"),
				Precise: y.Lines[y.MustLookupIndex("nagase1")].PortB.Length,
				Port:    layout.PortB,
			},
			BeforeFilled: true,
		}
		ca.After.Port = layout.PortA
		cb := GuideFence(y, &Train{
			Path:         &path,
			TrailerBack:  0,
			TrailerFront: 0,
		})
		if !cmp.Equal(ca, cb) {
			t.Log(cmp.Diff(ca, cb))
			t.Fatalf("diff")
		}
	}
	{
		ca := Constraint{
			Path:        path,
			After:       y.MustOffsetToPosition(path, 0),
			AfterFilled: true,
			Before: layout.Position{
				LineI:   y.MustLookupIndex("mitouc2"),
				Precise: y.Lines[y.MustLookupIndex("mitouc2")].PortB.Length,
				Port:    layout.PortB,
			},
			BeforeFilled: true,
		}
		ca.After.Port = layout.PortA
		cb := GuideFence(y, &Train{
			Path:         &path,
			TrailerBack:  0,
			TrailerFront: 1,
		})
		if !cmp.Equal(ca, cb) {
			t.Log(cmp.Diff(ca, cb))
			t.Fatalf("diff")
		}
	}
}
