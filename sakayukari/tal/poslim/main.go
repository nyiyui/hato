package poslim

import (
	"fmt"
	"sync"

	"golang.org/x/exp/slices"
	"nyiyui.ca/hato/sakayukari/tal"
	"nyiyui.ca/hato/sakayukari/tal/cars"
	"nyiyui.ca/hato/sakayukari/tal/layout"
)

type Assertion struct {
	TrainI          int
	TrainGeneration int
	After           layout.Offset
	AfterFilled     bool
	At              layout.Offset
	AtFilled        bool
	Before          layout.Offset
	BeforeFilled    bool
}

type Report struct {
	Trains []ReportTrain
}

type ReportTrain struct {
	// TODO: say if the train is before or after this point
	TrainI          int
	TrainGeneration int
	Position        layout.Position
	PositionType    PositionType
}

type PositionType int

const (
	PositionTypeInvalid PositionType = iota
	PositionTypeAfter
	PositionTypeAt
	PositionTypeBefore
	PositionTypeUnrelated // e.g. on different line than the train's path
)

type Witness interface {
	isWitness()
}

// PositionLimit calculates limits of trains from witnesses and current info from Guide.
type PositionLimit struct {
	g              *tal.Guide // constant
	witnesses      []*Witness // safe to replace
	cars           cars.Data  // safe to replace
	assertions     []Assertion
	assertionsLock sync.Mutex
	notifiees      []chan Report
	notifieesLock  sync.Mutex
}

func New(guide *tal.Guide, cars cars.Data) *PositionLimit {
	return &PositionLimit{
		g:    guide,
		cars: cars,
	}
}

func (pl *PositionLimit) AddNotifiee(notifiee chan []Assertion) {
	pl.notifieesLock.Lock()
	defer pl.notifieesLock.Unlock()
	pl.notifiees = append(pl.notifiees, notifiee)
}

func (pl *PositionLimit) RemoveNotifiee(notifiee chan []Assertion) {
	pl.notifieesLock.Lock()
	defer pl.notifieesLock.Unlock()
	panic("TODO")
}

func (pl *PositionLimit) notify() {
	pl.notifieesLock.Lock()
	defer pl.notifieesLock.Unlock()
	for _, notifiee := range pl.notifiees {
		notifiee <- pl.assertions
	}
}

func (pl *PositionLimit) NewRFIDWitness(pos layout.Position) *RFIDWitness {
	return &RFIDWitness{
		pl:  pl,
		pos: pos,
	}
}

func (pl *PositionLimit) report(r Report) {
	for _, rt := range r {
		pl.reportTrain(rt)
	}
}

func (pl *PositionLimit) reportTrain(rt ReportTrain) {
	pl.assertionsLock.Lock()
	defer pl.assertionsLock.Unlock()
	i := slices.IndexFunc(pl.assertions, func(a Assertion) bool { return a.TrainI == rt.TrainI })
	if i == -1 {
		i = len(pl.assertions)
		pl.assertions = append(pl.assertions, Assertion{
			TrainI:          rt.TrainI,
			TrainGeneration: rt.TrainGeneration,
		})
	}
	a := pl.assertions[i]
	if a.TrainGeneration > rt.TrainGeneration {
		// this ReportTrain is old
		return
	}
	a.TrainGeneration = rt.TrainGeneration
	switch rt.PositionType {
	case PositionTypeAfter:
		if (a.AfterFilled && rt.Position > a.After) || !a.AfterFilled {
			a.After = rt.Position
			a.AfterFilled = true
		}
	case PositionTypeAt:
		a.At = rt.Position
		a.AtFilled = true
	case PositionTypeBefore:
		if (a.BeforeFilled && rt.Position < a.Before) || !a.BeforeFilled {
			a.Before = rt.Position
			a.BeforeFilled = true
		}
	case PositionTypeUnrelated:
		// no info to add to Assertion
		return
	default:
		panic(fmt.Sprintf("invalid PositionType %d", rt.PositionType))
	}
	pl.assertions[i] = a
	pl.notify()
}
