package tal

import (
	"errors"
	"fmt"
	"time"

	"nyiyui.ca/hato/sakayukari/tal/layout"
)

type Constraint struct {
	Issued       time.Time
	Path         layout.FullPath
	After        layout.Position
	AfterFilled  bool
	Before       layout.Position
	BeforeFilled bool
}

func (c Constraint) check(y *layout.Layout) {
	if c.AfterFilled && c.BeforeFilled {
		after := y.PositionToOffset(c.Path, c.After)
		before := y.PositionToOffset(c.Path, c.Before)
		if after > before {
			panic(fmt.Sprintf("after > before: %#v", c))
		}
	}
}

func GuideFence(y *layout.Layout, t *Train) Constraint {
	var c Constraint
	c.Path = t.Path.Clone()

	{
		lp := t.Path.AtIndex(t.TrailerBack - 1)
		port := y.GetPort(lp)
		c.AfterFilled = true
		c.After.LineI = lp.LineI
		c.After.Port = lp.PortI
		c.After.Precise = port.Length
	}

	{
		lp := t.Path.Follows[t.TrailerFront]
		port := y.GetPort(lp)
		c.BeforeFilled = true
		c.Before.LineI = lp.LineI
		c.Before.Port = lp.PortI
		c.Before.Precise = port.Length
	}
	c.check(y)
	return c
}

func MinimumConstraint(y *layout.Layout, cs []Constraint) (Constraint, error) {
	if len(cs) == 0 {
		panic("must have at least 1 constraint given")
	}
	if len(cs) == 1 {
		return cs[0], nil
	}
	a := cs[0]
	for i := range cs {
		if i == 0 {
			continue
		}
		if !a.Path.Equal(cs[i].Path) {
			return Constraint{}, errors.New("paths not equal")
		}
	}
	path := a.Path

	var after int64 = -1
	var before int64 = -1
	for _, c := range cs {
		if c.AfterFilled {
			after2 := y.PositionToOffset(path, c.After)
			if after2 > after || after == -1 {
				after = after2
			}
		}
		if c.BeforeFilled {
			before2 := y.PositionToOffset(path, c.Before)
			if before2 < before || before == -1 {
				before = before2
			}
		}
	}
	var c2 Constraint
	c2.Path = path.Clone()
	if after != -1 {
		c2.AfterFilled = true
		c2.After = y.MustOffsetToPosition(path, after)
	}
	if before != -1 {
		c2.BeforeFilled = true
		c2.Before = y.MustOffsetToPosition(path, before)
	}
	return c2, nil
}

func FitInConstraint(y *layout.Layout, c Constraint, p Position) Position {
	c.check(y)
	offset := y.PositionToOffset(c.Path, p)
	if c.AfterFilled {
		after := y.PositionToOffset(c.Path, c.After)
		if offset < after {
			offset = after
		}
	}
	if c.BeforeFilled {
		before := y.PositionToOffset(c.Path, c.Before)
		if offset > before {
			offset = before
		}
	}
	return y.MustOffsetToPosition(c.Path, offset)
}
