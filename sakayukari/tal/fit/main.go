package fit

import "nyiyui.ca/hato/sakayukari/tal/poslim"

type Fit struct {
	g  *Guide
	pl *poslim.PositionLimit
}

func New(g *Guide, pl *poslim.PositionLimit) *Fit {
	f := &Fit{
		g:  g,
		pl: pl,
	}
	return f
}
func (f *Fit) Start() error {
	ch := make(chan []poslim.Assertion)
	f.pl.AddNotifiee(ch)
	defer f.pl.RemoveNotifiee(ch)
	for as := range ch {
	}
}
