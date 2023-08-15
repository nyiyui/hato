package tal

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/exp/slices"
	. "nyiyui.ca/hato/sakayukari"
	"nyiyui.ca/hato/sakayukari/conn"
	"nyiyui.ca/hato/sakayukari/tal/cars"
	"nyiyui.ca/hato/sakayukari/tal/layout"
)

type LineID = layout.LineID
type LinePort = layout.LinePort

const idlePower = 15
const switchPower = 192

type GuideConf struct {
	Layout        *layout.Layout
	Model         ActorRef
	Actors        map[LineID]ActorRef
	actorsReverse map[ActorRef]conn.Id
	Cars          cars.Data
	// Virtual disables serial commands to lines.
	Virtual bool
}

type TrainState int

const (
	// TrainStateNextAvail means the next line is available. The train should move to the next line.
	TrainStateNextAvail TrainState = 1
	// TrainStateNextLocked means the next line is locked by another train. The train should stop and wait at its current position, unless a precise attitude is available. If a precise attitude is available, it should stop without entering the next line.
	TrainStateNextLocked TrainState = 2
)

// TrainMode represents which of line or precise.
type TrainMode int

const (
	TrainModeLine TrainMode = iota
	TrainModePrecise
)

func (m TrainMode) String() string {
	switch m {
	case TrainModeLine:
		return "mode-line"
	case TrainModePrecise:
		return "mode-precise"
	default:
		return fmt.Sprintf("TrainMode unknown %d", m)
	}
}

type FormOrient int

const (
	FormOrientA FormOrient = iota + 1
	FormOrientB
)

// Flip returns the opposite orientation (A → B, B → A).
// If f is not A or B, this function panics.
func (f FormOrient) Flip() FormOrient {
	switch f {
	case FormOrientA:
		return FormOrientB
	case FormOrientB:
		return FormOrientA
	default:
		panic(fmt.Sprintf("invalid FormOrient %d", f))
	}
}

func (f FormOrient) String() string {
	switch f {
	case FormOrientA:
		return "fA"
	case FormOrientB:
		return "fB"
	default:
		return fmt.Sprintf("FormOrient_invalid_%d", f)
	}
}

type Train struct {
	// Generation is incremented whenever any non-dynamic field is changed.
	Generation int
	// TODO: GenerationChanges (e.g. did power, orient change?)

	// Power supplied directly to soyuu-line (when moving)
	Power           int
	noPowerSupplied bool

	// Mode is which mode this train is in. See TrainMode for details.
	// This is a dynamic field.
	Mode TrainMode
	// PreciseAttitudeOnly is true when TrailerBack, TrailerFront, CurrentBack, and CurrentFront hold no meaning, and all locking and energization is done using formation info and tal-model.
	// If PreciseAttitudeOnly is true when only TrailerBack and TrailerFront is determined using tal-modle (CurrentBack and CurrentFront still hold meaning, as the base part which is always energized).
	PreciseAttitudeOnly bool
	latestAttitude      Attitude

	// This is a dynamic field.
	TrailerBack int
	// This is a dynamic field.
	TrailerFront int
	// CurrentBack is the path index of the last car's occupying line.
	// Must always be larger than 0.
	// This is a dynamic field.
	CurrentBack int
	// CurrentFront is the path index of the first car's occupying line.
	// Must always be larger than 0.
	// This is a dynamic field.
	CurrentFront int
	// Path is the Path of outgoing LinePorts until the goal.
	// This should be generated by FullPathTo, and must contain on index 0 a LinePort with the same line as index 1 and a opposite port to index 1's LinePort.
	Path *layout.FullPath
	// This is a dynamic field.
	State TrainState

	FormI uuid.UUID
	// Orient shows which side (side A or B) the front of the train (c.f. CurrentFront etc).
	Orient FormOrient
}

// nextUnsafe returns the path index of the next LinePort.
// Note: this does check if this train has a next available, and panics if next is not available.
func (t *Train) next() int {
	if t.State != TrainStateNextAvail {
		panic("next() called when not trainStateNextAvail")
	}
	return t.nextUnsafe()
}

// nextUnsafe returns the path index of the next LinePort.
// Note: this does not check if this train has a next available.
func (t *Train) nextUnsafe() int {
	return t.TrailerFront + 1
}

func (t *Train) String() string {
	b := new(strings.Builder)
	fmt.Fprintf(b, "power%d %d-%d", t.Power, t.CurrentBack, t.CurrentFront)
	switch t.State {
	case TrainStateNextAvail:
		fmt.Fprintf(b, "→%d", t.next())
	case TrainStateNextLocked:
		fmt.Fprintf(b, "L")
	}
	fmt.Fprintf(b, " S%sF", t.Path.Start)
	for _, lp := range t.Path.Follows {
		fmt.Fprintf(b, " %s", lp)
	}
	return b.String()
}

type guide struct {
	actor      Actor
	conf       GuideConf
	trains     []Train
	lineStates []LineStates
	y          *layout.Layout
}

type LineStates struct {
	Taken           bool
	TakenBy         int
	PowerActor      ActorRef
	Power           uint8
	SwitchActor     ActorRef
	SwitchState     SwitchState
	nextSwitchState SwitchState
}

func Guide(conf GuideConf) Actor {
	if conf.Cars.Forms == nil {
		panic("conf.Cars required")
	}
	a := Actor{
		Comment:  "tal-guide",
		InputCh:  make(chan Diffuse1),
		OutputCh: make(chan Diffuse1),
		Inputs:   make([]ActorRef, 0),
		Type: ActorType{
			Input:       true,
			LinearInput: true,
			Output:      true,
		},
	}
	a.Inputs = append(a.Inputs, conf.Model)
	for _, l := range conf.Layout.Lines {
		a.Inputs = append(a.Inputs, conf.Actors[l.PowerConn])
		if l.IsSwitch() {
			a.Inputs = append(a.Inputs, conf.Actors[l.SwitchConn])
		}
	}
	conf.actorsReverse = map[ActorRef]conn.Id{}
	for li, ar := range conf.Actors {
		conf.actorsReverse[ar] = li.Conn
	}
	g := guide{
		conf:       conf,
		actor:      a,
		trains:     make([]Train, 0),
		lineStates: make([]LineStates, len(conf.Layout.Lines)),
		y:          conf.Layout,
	}
	t1 := Train{
		Power:        60,
		CurrentBack:  0,
		CurrentFront: 0,
		State:        TrainStateNextAvail,
		FormI:        uuid.MustParse("e5f6bb45-0abe-408c-b8e0-e2772f3bbdb0"),
		//FormI: uuid.MustParse("2fe1cbb0-b584-45f5-96ec-a9bfd55b1e91"),
		//FormI:  uuid.MustParse("7b920d78-0c1b-49ef-ab2e-c1209f49bbc6"),
		Orient: FormOrientA,
	}
	path := g.y.MustFullPathTo(LinePort{g.y.MustLookupIndex("A"), layout.PortB}, LinePort{g.y.MustLookupIndex("C"), layout.PortA})
	t1.Path = &path
	log.Printf("t1.Path %#v", path)
	g.trains = append(g.trains, t1)

	go g.loop()
	return a
}

func (g *guide) calculateTrailers(t *Train) {
	sideA, sideB := g.conf.Cars.Forms[t.FormI].TrailerLength()
	if sideA == 0 && sideB == 0 {
		t.TrailerBack = t.CurrentBack
		t.TrailerFront = t.CurrentFront
		return
	}
	trailerBack, trailerFront := t.CurrentBack, t.CurrentFront
	backPossible := true
	// back is the length from port A of CurrentBack to the backside of the trailers.
	var back int64
	frontPossible := true
	// front is the length from port A of CurrentFront to the frontside of the trailers.
	var front int64
	switch t.Mode {
	case TrainModePrecise:
		att := t.latestAttitude
		var path []LinePort
		switch t.Orient {
		case FormOrientA:
			// traverse front → back
			reversed := g.y.ReverseFullPath(*t.Path).Follows
			i := slices.IndexFunc(t.Path.Follows, func(lp LinePort) bool { return lp.LineI == att.Position.LineI })
			path = reversed[i:]
			trailerFront = i
		case FormOrientB:
			// traverse back → front
			i := slices.IndexFunc(t.Path.Follows, func(lp LinePort) bool { return lp.LineI == att.Position.LineI })
			path = t.Path.Follows[i:]
			trailerBack = i
		}
		form := g.conf.Cars.Forms[t.FormI]
		side2, ok := g.y.Traverse(path, int64(form.Length))
		if !ok {
			// Take the longest possible place, as overflow means we went over the former.
			switch t.Orient {
			case FormOrientA:
				// traverse front → back
				trailerBack = 0
			case FormOrientB:
				// traverse back → front
				trailerFront = len(t.Path.Follows) - 1
			}
		} else {
			switch t.Orient {
			case FormOrientA:
				// traverse front → back
				trailerBack = slices.IndexFunc(t.Path.Follows, func(lp LinePort) bool { return lp.LineI == side2.LineI })
			case FormOrientB:
				// traverse back → front
				trailerFront = slices.IndexFunc(t.Path.Follows, func(lp LinePort) bool { return lp.LineI == side2.LineI })
			}
		}
		if !t.PreciseAttitudeOnly {
			if trailerFront < t.CurrentFront {
				trailerFront = t.CurrentFront
			}
			if trailerBack > t.CurrentBack {
				trailerBack = t.CurrentBack
			}
		}
		log.Printf("TrainModePrecise trailer %d-%d", trailerBack, trailerFront)
		log.Printf("TrainModePrecise current %d-%d", t.CurrentBack, t.CurrentFront)
	case TrainModeLine:
		log.Printf("form %#v", g.conf.Cars.Forms[t.FormI])
		log.Printf("formI %#v", t.FormI)
		sideA, sideB := g.conf.Cars.Forms[t.FormI].TrailerLength()
		log.Printf("sideA %d", sideA)
		log.Printf("sideB %d", sideB)
		switch t.Orient {
		case FormOrientA:
			front, back = sideA, sideB
		case FormOrientB:
			front, back = sideB, sideA
		}
		if t.CurrentBack == 0 {
			backPossible = false
		} else {
			behindBack := t.Path.Follows[t.CurrentBack-1]
			// backside is the backmost port of CurrentBack.
			backside := g.y.GetPort(behindBack).Conn()
			if backside.PortI == -1 {
				// I guess there's not much of a point now…
			} else if backside.PortI != layout.PortA {
				back += int64(g.y.GetPort(backside).Length)
			}
		}
		if t.CurrentFront == len(t.Path.Follows)-1 {
			frontPossible = false
		} else if lp := t.Path.Follows[t.CurrentFront]; lp.PortI != layout.PortA {
			_, p := g.y.GetLinePort(lp)
			front += int64(p.Length)
		}
		log.Printf("back %d", back)
		log.Printf("front %d", front)
		_ = backPossible
		if frontPossible {
			follows := t.Path.Follows[t.CurrentFront:]
			pos, ok := g.y.Traverse(follows, front)
			if !ok {
				log.Printf("train: trailer overrun (front)")
				trailerFront = len(t.Path.Follows) - 1
			} else {
				trailerFront = slices.IndexFunc(t.Path.Follows, func(lp LinePort) bool { return lp.LineI == pos.LineI })
				log.Printf("new CurrentFront = %d %#v", trailerFront, t.Path.Follows[trailerFront])
			}
		}
		if backPossible {
			follows := g.y.ReverseFullPath(*t.Path).Follows
			log.Printf("t.Path %#v", t.Path)
			log.Printf("follows1 %#v", follows)
			follows = follows[slices.IndexFunc(follows, func(lp LinePort) bool { return lp.LineI == t.Path.Follows[t.CurrentBack].LineI }):]
			log.Printf("follows2 %#v", follows)
			pos, ok := g.y.Traverse(follows, back)
			if !ok {
				log.Printf("train: trailer overrun (back)")
				trailerBack = 0
			} else {
				trailerBack = slices.IndexFunc(t.Path.Follows, func(lp LinePort) bool { return lp.LineI == pos.LineI })
				log.Printf("new CurrentBack = %d %#v", trailerBack, t.Path.Follows[trailerBack])
			}
		}
	}
	t.TrailerBack = trailerBack
	t.TrailerFront = trailerFront
}

func (g *guide) handleValCurrent(diffuse Diffuse1, cur conn.ValCurrent) {
	ci, ok := g.conf.actorsReverse[diffuse.Origin]
	if !ok {
		log.Printf("unknown conn for actor %s", diffuse.Origin)
		return
	}
	//log.Printf("=== diffuse from %s: %s", ci, cur)
	for ti := range g.trains {
		for _, inner := range cur.Values {
			t := g.trains[ti]
			if t.noPowerSupplied {
				continue
			}
			// sync t.state etc
			g.syncLocks(ti)
			t = g.trains[ti]

			cb := g.y.Lines[t.Path.Follows[t.CurrentBack].LineI]
			if ci == cb.PowerConn.Conn && inner.Line == cb.PowerConn.Line && !inner.Flow {
				if t.CurrentBack >= t.CurrentFront {
					// this can happen e.g. when the train is at 0-0→1 and then the 0th line becomes 0 (e.g. A0, B0)
					goto NoCurrentBack
				}
				nextI := t.Path.Follows[t.CurrentBack].LineI
				g.unlock(nextI)
				g.apply(&t, t.CurrentBack, 0)
				t.CurrentBack++
				g.calculateTrailers(&t)
				//log.Printf("=== currentBack succession: %d", t.CurrentBack)
				g.publishChange(ti, ChangeTypeCurrentBack)
			}
		NoCurrentBack:
			cf := g.y.Lines[t.Path.Follows[t.CurrentFront].LineI]
			if ci == cf.PowerConn.Conn && inner.Line == cf.PowerConn.Line && !inner.Flow {
				if t.CurrentFront == 0 {
					//log.Printf("=== currentFront regression (ignore): %d", t.CurrentFront)
					goto NoCurrentFront
				}
				if t.CurrentFront <= t.CurrentBack {
					// this can happen e.g. when the train is at 1-1→2 and then the 1st line becomes 0 (e.g. A0, B0) (currentBack moving to 0 is prevented by an if for currentBack)
					//log.Printf("=== currentFront regression (ignore as currentFront <= currentBack): %d", t.CurrentFront)
					goto NoCurrentFront
				}
				nextI := t.Path.Follows[t.CurrentFront].LineI
				g.unlock(nextI)
				g.apply(&t, t.CurrentFront, 0)
				t.CurrentFront--
				g.calculateTrailers(&t)
				g.publishChange(ti, ChangeTypeCurrentFront)
				//log.Printf("=== currentFront regression: %d", t.CurrentFront)
			}
		NoCurrentFront:
			if t.State == TrainStateNextAvail {
				// if t.state ≠ trainStateNextAvail, t.next could be out of range
				cf := g.y.Lines[t.Path.Follows[t.next()].LineI]
				if ci == cf.PowerConn.Conn && inner.Line == cf.PowerConn.Line && inner.Flow {
					t.CurrentFront++
					g.calculateTrailers(&t)
					g.publishChange(ti, ChangeTypeCurrentFront)
					//log.Printf("=== next succession: %d", t.CurrentFront)
				}
			}
			g.trains[ti] = t
		}
		// TODO: check if the train derailed, was removed, etc (come up with a heuristic)
		// TODO: check for regressions
		// TODO: check for overruns (is this possible?)
		//log.Printf("postshow: %s", &g.trains[ti])
	}
	g.publishSnapshot()
	for ti := range g.trains {
		g.wakeup(ti, "post-handleValCurrent")
		//log.Printf("postwakeup: %s", &g.trains[ti])
	}
	g.publishSnapshot()
}

func (g *guide) handleAttitude(att Attitude) {
	t := &g.trains[att.TrainI]
	if att.TrainGeneration < t.Generation {
		return
	}
	oldMode := t.Mode
	if att.PositionKnown {
		t.Mode = TrainModePrecise
	} else {
		t.Mode = TrainModeLine
	}
	t.latestAttitude = att
	if oldMode != t.Mode {
		g.wakeup(att.TrainI, fmt.Sprintf("change mode %s → %s", oldMode, t.Mode))
	}
}

// reason is only for debugging.
func (g *guide) wakeup(ti int, reason string) {
	log.Printf("wakeup %d", ti)
	log.Printf("wakeup %#v", g.trains[ti])
	log.Printf("wakeup %#v", g.trains[ti].Path)
	g.check(ti)
	g.syncLocks(ti)
	t := g.trains[ti]
	g.reify(ti, &t)
	g.trains[ti] = t
}

func (g *guide) check(ti int) {
	t := g.trains[ti]
	if t.Power < 0 {
		panic(fmt.Sprintf("TrainI %d: negative power: %#v", ti, t))
	}
}

func (g *guide) loop() {
	time.Sleep(1 * time.Second)
	for ti := range g.trains {
		g.wakeup(ti, "init")
	}
	g.publishSnapshot()
	for diffuse := range g.actor.InputCh {
		if diffuse.Origin == g.conf.Model {
			switch val := diffuse.Value.(type) {
			case Attitude:
				g.handleAttitude(val)
			}
		}
		switch val := diffuse.Value.(type) {
		case GuideTrainUpdate:
			log.Printf("diffuse GuideTrainUpdate %d %#v", val.TrainI, val.Train)
			orig := g.trains[val.TrainI]
			if val.Train.Power == -1 {
				val.Train.Power = orig.Power
			}
			if val.Train.Mode != 0 {
				panic("Train.Mode cannot be set.")
			}
			val.Train.Mode = orig.Mode
			if val.Train.CurrentBack == layout.BlankLineI {
				val.Train.CurrentBack = orig.CurrentBack
			}
			if val.Train.CurrentFront == layout.BlankLineI {
				val.Train.CurrentFront = orig.CurrentFront
			}
			if val.Train.Path == nil {
				val.Train.Path = orig.Path
			}
			if val.Train.State == 0 {
				val.Train.State = orig.State
			}
			if val.Train.FormI == (uuid.UUID{}) {
				val.Train.FormI = orig.FormI
			}
			backSame := val.Train.Path.Follows[val.Train.CurrentBack] == orig.Path.Follows[orig.CurrentBack]
			frontSame := val.Train.Path.Follows[val.Train.CurrentFront] == orig.Path.Follows[orig.CurrentFront]
			if backSame != frontSame {
				backA := val.Train.Path.Follows[val.Train.CurrentBack]
				backB := orig.Path.Follows[orig.CurrentBack]
				frontA := val.Train.Path.Follows[val.Train.CurrentFront]
				frontB := orig.Path.Follows[orig.CurrentFront]
				log.Printf("backA %#v", backA)
				log.Printf("backB %#v", backB)
				log.Printf("frontA %#v", frontA)
				log.Printf("frontB %#v", frontB)
				panic("The two lines pointed to by CurrentFront and CurrentBack must be the same two, in any order (they can be swapped).")
			}
			if val.Train.Orient == 0 {
				if backSame && frontSame {
					val.Train.Orient = orig.Orient
				} else if !backSame && !frontSame {
					val.Train.Orient = orig.Orient.Flip()
				} else {
					panic("unreachable")
				}
			}
			val.Train.Generation = orig.Generation + 1
			log.Printf("GuideTrainUpdate %#v", val.Train)
			log.Printf("GuideTrainUpdate.Path %#v", val.Train.Path)
			g.calculateTrailers(&val.Train)
			g.trains[val.TrainI] = val.Train
			g.wakeup(val.TrainI, "GuideTrainUpdate")
		case conn.ValCurrent:
			g.handleValCurrent(diffuse, val)
		case conn.ValShortNotify:
			c := g.conf.actorsReverse[diffuse.Origin]
			li := slices.IndexFunc(g.y.Lines, func(l layout.Line) bool { return l.SwitchConn == (LineID{Conn: c, Line: val.Line}) })
			if li == -1 {
				panic(fmt.Sprintf("no line found for ValShortNotify %#v", diffuse))
			}
			ls := g.lineStates[li]
			log.Printf("lineState %#v", ls)
			if !ls.Taken {
				panic(fmt.Sprintf("ValShortNotify for non-taken line %d %#v", li, ls))
			}
			g.lineStates[li].SwitchState = ls.nextSwitchState
			g.lineStates[li].nextSwitchState = 0
			log.Printf("wakeup %d %s", ls.TakenBy, &g.trains[ls.TakenBy])
			g.wakeup(ls.TakenBy, "ValShortNotify")
		}
		g.publishSnapshot()
	}
}

func reverse[S ~[]E, E any](s S) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

func (g *guide) idlePower(ti int) int {
	t := g.trains[ti]
	f, ok := g.conf.Cars.Forms[t.FormI]
	if !ok {
		return idlePower
	}
	if f.BaseVelocity == nil {
		return idlePower
	}
	m := f.BaseVelocity.M
	b := f.BaseVelocity.B
	return int(conn.AbsClampPower(int(-b / m)))
}

func (g *guide) reify(ti int, t *Train) {
	power := t.Power
	stop := false
	max := t.TrailerFront
	if t.State == TrainStateNextAvail {
		max += 1
	}
	for i := t.TrailerBack; i <= max; i++ {
		if g.lineStates[t.Path.Follows[i].LineI].SwitchState == SwitchStateUnsafe {
			log.Printf("=== STOP UNSAFE")
			stop = true
			power = g.idlePower(ti)
			break
		}
	}
	if t.State == TrainStateNextLocked {
		log.Printf("=== TrainStateNextLocked")
		log.Printf("t %#v", t)
		stop = true
	}
	if stop {
		power = g.idlePower(ti)
	}
	log.Printf("REIFY: %d %s", power, t)
	t.noPowerSupplied = power < 15
	for i := t.TrailerBack; i <= t.TrailerFront; i++ {
		g.applySwitch(ti, t, i)
		g.apply(t, i, power)
	}
	if t.State == TrainStateNextAvail {
		g.applySwitch(ti, t, t.next())
		g.apply(t, t.next(), power)
	}
}

func (g *guide) applySwitch(ti int, t *Train, pathI int) {
	li := t.Path.Follows[pathI].LineI
	pi := t.Path.Follows[pathI].PortI
	//log.Printf("=== applySwitch path%d %s", pathI, g.y.Lines[li].Comment)
	if g.y.Lines[li].SwitchConn == (LineID{}) {
		// no switch here
		return
	}
	var targetState SwitchState
	if pi == 0 {
		// merging, so check switch is in the right direction
		var lp LinePort
		if pathI == 0 {
			lp = t.Path.Start
		} else {
			lp = t.Path.Follows[pathI-1]
		}
		p := g.y.Lines[lp.LineI].GetPort(lp.PortI)
		switch p.ConnP { // p.ConnP is what the line connecting to the merging switch connects to
		case layout.PortA:
			panic("merging from port A to port A! Cannot change direction suddenly")
		case layout.PortB:
			// The train goes from port B to A
			targetState = SwitchStateB
		case layout.PortC:
			// The train goes from port C to A
			targetState = SwitchStateC
		default:
			panic("invalid ConnP")
		}
	} else {
		if pi == 1 && g.lineStates[li].SwitchState == SwitchStateB {
			return
		} else if pi == 2 && g.lineStates[li].SwitchState == SwitchStateC {
			return
		}
		switch pi {
		case 1:
			targetState = SwitchStateB
		case 2:
			targetState = SwitchStateC
		default:
			panic(fmt.Sprintf("invalid pi %d", pi))
		}
	}
	if g.lineStates[li].SwitchState == targetState {
		return
	}
	if g.lineStates[li].SwitchState == SwitchStateUnsafe {
		// already switching
		return
	}
	g.lineStates[li].SwitchState = SwitchStateUnsafe
	g.lineStates[li].nextSwitchState = targetState

	//log.Printf("applySwitch")
	d := Diffuse1{
		Origin: g.conf.Actors[g.y.Lines[li].SwitchConn],
		Value: conn.ReqSwitch{
			Line:      g.y.Lines[li].SwitchConn.Line,
			Direction: targetState == SwitchStateB,
			// true  when targetState is B
			// false when targetState is C
			Power:    switchPower,
			Duration: 1000,
		},
	}
	//log.Printf("diffuse %#v", d)
	if g.conf.Virtual {
		return
	}
	g.actor.OutputCh <- d
}

func (g *guide) apply(t *Train, pathI int, power int) {
	pi := t.Path.Follows[pathI].PortI
	li := t.Path.Follows[pathI].LineI
	l := g.y.Lines[li]
	rl := conn.ReqLine{
		Line: l.PowerConn.Line,
		// NOTE: reversed for now as the layout is reversed (bodge)
		// false if port A, true if port B or C
		Power: conn.AbsClampPower(power),
	}
	g.lineStates[li].Power = rl.Power
	rl.Direction = l.GetPort(pi).Direction
	// TODO: fix direction to follow layout.Layout rules
	log.Printf("apply %s %s to %s", t, rl, g.conf.Actors[l.PowerConn])
	if g.conf.Virtual {
		log.Printf("apply2 virtual %s", rl)
		return
	}
	g.actor.OutputCh <- Diffuse1{
		Origin: g.conf.Actors[l.PowerConn],
		Value:  rl,
	}
	log.Printf("apply2 %s", rl)
}

// syncLocks verifies locking of all currents and next (if next is available) of a train.
func (g *guide) syncLocks(ti int) {
	t := g.trains[ti]
	defer func() { g.trains[ti] = t }()
	for i := t.TrailerBack; i <= t.TrailerFront; i++ {
		ok := g.lock(t.Path.Follows[i].LineI, ti)
		if !ok {
			panic(fmt.Sprintf("train %s currents %d: locking failed", &t, i))
		}
	}
	if t.TrailerFront == len(t.Path.Follows)-1 {
		// end of path
		t.State = TrainStateNextLocked
	} else {
		ok := g.lock(t.Path.Follows[t.nextUnsafe()].LineI, ti)
		if ok {
			t.State = TrainStateNextAvail
		} else {
			t.State = TrainStateNextLocked
			log.Printf("train %d: failed to lock %d", ti, t.nextUnsafe())
		}
	}
}

// lockSync tries to lock all LineIs in lis. If any fails, it returns ok = false.
func (g *guide) lockSync(lis []layout.LineI, ti int) (ok bool) {
	for _, li := range lis {
		if g.lineStates[li].Taken {
			if g.lineStates[li].TakenBy != ti {
				return false
			} else {
				return true
			}
		}
		//log.Printf("LOCK %d(%s) by %d", li, g.y.Lines[li].Comment, ti)
		g.lineStates[li].Taken = true
		g.lineStates[li].TakenBy = ti
	}
	return true
}

func (g *guide) lock(li layout.LineI, ti int) (ok bool) {
	if g.lineStates[li].Taken {
		if g.lineStates[li].TakenBy != ti {
			return false
		} else {
			return true
		}
	}
	//log.Printf("LOCK %d(%s) by %d", li, g.y.Lines[li].Comment, ti)
	g.lineStates[li].Taken = true
	g.lineStates[li].TakenBy = ti
	return true
}

func (g *guide) unlock(li layout.LineI) {
	//log.Printf("UNLOCK %d(%s) by %d", li, g.y.Lines[li].Comment, g.lineStates[li].TakenBy)
	g.lineStates[li].Taken = false
	g.lineStates[li].TakenBy = -1
	// TODO: maybe do wakeup for all trains that match (instead of the dumb for loop in guide.single())
}

type GuideTrainUpdate struct {
	TrainI int
	// Train has the updated values. Currently, only Train.Power is updated.
	// TODO: allow updating Train.Path
	Train Train
}

func (gtu GuideTrainUpdate) String() string {
	return fmt.Sprintf("GuideTrainUpdate %d %#v", gtu.TrainI, gtu.Train)
}

type GuideSnapshot struct {
	Trains     []Train
	Layout     *layout.Layout
	LineStates []LineStates
}

func (gs GuideSnapshot) String() string {
	b := new(strings.Builder)
	b.WriteString("GuideSnapshot")
	for ti, t := range gs.Trains {
		fmt.Fprintf(b, "\n%d %s", ti, &t)
	}
	return b.String()
}

func (g *guide) snapshot() GuideSnapshot {
	gs := GuideSnapshot{Trains: g.trains, Layout: g.conf.Layout, LineStates: g.lineStates}
	buf := new(bytes.Buffer)
	err := gob.NewEncoder(buf).Encode(gs)
	if err != nil {
		panic(fmt.Sprintf("snapshot: encode: %s", err))
	}
	var res GuideSnapshot
	err = gob.NewDecoder(buf).Decode(&res)
	if err != nil {
		panic(fmt.Sprintf("snapshot: decode: %s", err))
	}
	return res
}

func (g *guide) publishSnapshot() {
	g.actor.OutputCh <- Diffuse1{Value: g.snapshot()}
}

type GuideChange struct {
	TrainI   int
	Type     ChangeType
	Snapshot GuideSnapshot
}

func (gc GuideChange) String() string {
	return fmt.Sprintf("%#v", gc)
}

type ChangeType int

const (
	ChangeTypeCurrentBack ChangeType = iota + 1
	ChangeTypeCurrentFront
)

func (g *guide) publishChange(ti int, ct ChangeType) {
	//log.Printf("=== publishChange %d %#v", ti, ct)
	g.actor.OutputCh <- Diffuse1{Value: GuideChange{
		TrainI:   ti,
		Type:     ct,
		Snapshot: g.snapshot(),
	}}
}
