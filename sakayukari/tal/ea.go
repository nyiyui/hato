package tal

import (
	"log"

	"golang.org/x/exp/slices"
	"nyiyui.ca/hato/sakayukari/tal/cars"
	"nyiyui.ca/hato/sakayukari/tal/layout"
)

type EAPass struct {
	// Attitudes is a list of expected attitudes in order from the start of the path to the end.
	Attitudes []ExpectedAttitude
	Passes    []bool
}

type ExpectedAttitude struct {
	Position layout.Position
	// TODO: need some way to determine which RFID sensor-tag pair it is
	//       tag is easy, how to identify sensor? ActorRef?
}

func (m *model) getExpectedAttitudes(ti int) []ExpectedAttitude {
	eas := make([]ExpectedAttitude, 0)
	t := m.latestGS.Trains[ti]
	f := m.conf.Cars.Forms[t.FormI]
	for _, p := range t.Path.Follows {
		for _, r := range m.conf.RFIDs {
			if r.Position.LineI == p.LineI {
				for ci, c := range f.Cars {
					if c.MifareID != (cars.MifareID{}) {
						// calculate TagOffset
						tagOffset := f.TagOffset(ci)
						_ = tagOffset
						switch t.Orient {
						case FormOrientA:
						case FormOrientB:
						default:
							panic("invalid Train.FormOrient")
						}
						eas = append(eas, ExpectedAttitude{r.Position})
					}
				}
			}
		}
	}
	return eas
}

// reverseIndex returns the index of the first occurrence of v in s,
// or -1 if not present.
func reverseIndex[S ~[]E, E comparable](s S, v E) int {
	for i := len(s) - 1; i >= 0; i-- {
		if v == s[i] {
			return i
		}
	}
	return -1
}

func (m *model) clampExpectedAttitudes(ti int, pos layout.Position) layout.Position {
	// find extremes
	eap := m.eaps[ti]
	t := m.latestGS.Trains[ti]
	backI := slices.Index(eap.Passes, true)
	frontI := reverseIndex(eap.Passes, true)
	if backI == -1 || frontI == -1 {
		log.Printf("blank EAPasses: %#v", eap)
		return pos
	}
	back, front := eap.Attitudes[backI].Position, eap.Attitudes[frontI].Position
	if backI > frontI {
		panic("backI > frontI")
	}
	y := m.latestGS.Layout
	offset := y.PositionToOffset(*t.Path, pos)
	backOffset := y.PositionToOffset(*t.Path, back)
	frontOffset := y.PositionToOffset(*t.Path, front)
	if backOffset > frontOffset {
		panic("backOffset > frontOffset")
	}
	if offset < backOffset {
		offset = backOffset
	} else if offset > frontOffset {
		offset = frontOffset
	}
	return y.OffsetToPosition(*t.Path, offset)
}

func (m *model) updateEAPasses() {
	init := false
	if m.eaps == nil {
		m.eaps = make([]EAPass, len(m.latestGS.Trains))
		init = true
	}
	for ti := range m.latestGS.Trains {
		eas := m.getExpectedAttitudes(ti)
		if init {
			m.eaps[ti] = EAPass{
				Attitudes: eas,
				Passes:    make([]bool, len(eas)),
			}
		} else {
			oldEAP := m.eaps[ti]
			eap := EAPass{
				Attitudes: eas,
				Passes:    make([]bool, len(eas)),
			}
			for i, att := range oldEAP.Attitudes {
				if oldEAP.Passes[i] {
					if i := slices.Index(eap.Attitudes, att); i != -1 {
						eap.Passes[i] = true
					}
				}
			}
			m.eaps[ti] = eap
		}
	}
}
