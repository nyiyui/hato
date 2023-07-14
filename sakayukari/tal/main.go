package tal

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"strings"
	"time"

	. "nyiyui.ca/hato/sakayukari"
	"nyiyui.ca/hato/sakayukari/conn"
	"nyiyui.ca/hato/sakayukari/tal/layout"
)

type LineID = layout.LineID
type LinePort = layout.LinePort

const idlePower = 15

// guide - uses line to move trains
// adjuster - adjusts power level etc

type GuideConf struct {
	Layout        *layout.Layout
	Actors        map[LineID]ActorRef
	actorsReverse map[ActorRef]conn.Id
}

type TrainState int

const (
	// TrainStateNextAvail means the next line is available. The train should move to the next line.
	TrainStateNextAvail TrainState = 1
	// TrainStateNextLocked means the next line is locked by another train. The train should stop and wait at its current position, unless a precise attitude is available. If a precise attitude is available, it should stop without entering the next line.
	TrainStateNextLocked TrainState = 2
)

type Train struct {
	// Power supplied directly to soyuu-line (when moving)
	Power           int
	noPowerSupplied bool

	// dynamic fields

	// CurrentBack is the path index of the last car's occupying line.
	CurrentBack int
	// CurrentFront is the path index of the first car's occupying line.
	CurrentFront int
	// Path is the Path of outgoing LinePorts until the goal.
	Path  []LinePort
	State TrainState
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
	return t.CurrentFront + 1
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
	for _, lp := range t.Path {
		fmt.Fprintf(b, " %s", lp)
	}
	return b.String()
}

type guide struct {
	actor      Actor
	conf       GuideConf
	trains     []Train
	lineStates []lineState
	y          *layout.Layout
}

type lineState struct {
	Taken           bool
	TakenBy         int
	PowerActor      ActorRef
	SwitchActor     ActorRef
	SwitchState     SwitchState
	nextSwitchState SwitchState
}

func Guide(conf GuideConf) Actor {
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
		lineStates: make([]lineState, len(conf.Layout.Lines)),
		y:          conf.Layout,
	}
	t1 := Train{
		Power:        70,
		CurrentBack:  0,
		CurrentFront: 0,
		State:        TrainStateNextAvail,
	}
	//t1.path = g.y.PathTo(g.y.MustLookupIndex("Y"), g.y.MustLookupIndex("W")) // reverse
	t1.Path = g.y.PathTo(g.y.MustLookupIndex("Y"), g.y.MustLookupIndex("X")) // normal
	{
		last := t1.Path[len(t1.Path)-1]
		p := g.y.Lines[last.LineI].GetPort(last.PortI)
		t1.Path = append(t1.Path, LinePort{LineI: p.ConnI, PortI: -1})
	}
	g.trains = append(g.trains, t1)

	go g.loop()
	return a
}

func (g *guide) handleValCurrent(diffuse Diffuse1, cur conn.ValCurrent) {
	ci, ok := g.conf.actorsReverse[diffuse.Origin]
	if !ok {
		log.Printf("unknown conn for actor %s", diffuse.Origin)
		return
	}
	log.Printf("=== diffuse from %s: %s", ci, cur)
	for ti, t := range g.trains {
		for _, inner := range cur.Values {
			if t.noPowerSupplied {
				continue
			}
			// sync t.state etc
			g.syncLocks(ti)
			t = g.trains[ti]

			cb := g.y.Lines[t.Path[t.CurrentBack].LineI]
			if ci == cb.PowerConn.Conn && inner.Line == cb.PowerConn.Line && !inner.Flow {
				if t.CurrentBack >= t.CurrentFront {
					// this can happen e.g. when the train is at 0-0→1 and then the 0th line becomes 0 (e.g. A0, B0)
					goto NoCurrentBack
				}
				nextI := t.Path[t.CurrentBack].LineI
				g.unlock(nextI)
				g.apply(&t, t.CurrentBack, 0)
				t.CurrentBack++
				log.Printf("=== currentBack succession: %d", t.CurrentBack)
			}
		NoCurrentBack:
			cf := g.y.Lines[t.Path[t.CurrentFront].LineI]
			if ci == cf.PowerConn.Conn && inner.Line == cf.PowerConn.Line && !inner.Flow {
				if t.CurrentFront == 0 {
					log.Printf("=== currentFront regression (ignore): %d", t.CurrentFront)
					goto NoCurrentFront
				}
				if t.CurrentFront <= t.CurrentBack {
					// this can happen e.g. when the train is at 1-1→2 and then the 1st line becomes 0 (e.g. A0, B0) (currentBack moving to 0 is prevented by an if for currentBack)
					log.Printf("=== currentFront regression (ignore as currentFront <= currentBack): %d", t.CurrentFront)
					goto NoCurrentFront
				}
				nextI := t.Path[t.CurrentFront].LineI
				g.unlock(nextI)
				g.apply(&t, t.CurrentFront, 0)
				t.CurrentFront--
				log.Printf("=== currentFront regression: %d", t.CurrentFront)
			}
		NoCurrentFront:
			if t.State == TrainStateNextAvail {
				// if t.state ≠ trainStateNextAvail, t.next could be out of range
				cf := g.y.Lines[t.Path[t.next()].LineI]
				if ci == cf.PowerConn.Conn && inner.Line == cf.PowerConn.Line && inner.Flow {
					t.CurrentFront++
					log.Printf("=== next succession: %d", t.CurrentFront)
					log.Printf("inner: %#v", inner)
					log.Printf("what: %s", &t)
				}
			}
		}
		// TODO: check if the train derailed, was removed, etc (come up with a heuristic)
		// TODO: check for regressions
		// TODO: check for overruns (is this possible?)
		g.trains[ti] = t
		log.Printf("postshow: %s", &t)
	}
	g.publishSnapshot()
	for ti := range g.trains {
		g.wakeup(ti)
		log.Printf("postwakeup: %s", &g.trains[ti])
	}
	g.publishSnapshot()
}

func (g *guide) wakeup(ti int) {
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
		g.wakeup(ti)
	}
	g.publishSnapshot()
	for diffuse := range g.actor.InputCh {
		switch val := diffuse.Value.(type) {
		case GuideTrainUpdate:
			log.Printf("diffuse GuideTrainUpdate")
			orig := g.trains[val.TrainI]
			if val.Train.Power == -1 {
				val.Train.Power = orig.Power
			}
			if val.Train.CurrentBack == -1 {
				val.Train.CurrentBack = orig.CurrentBack
			}
			if val.Train.CurrentFront == -1 {
				val.Train.CurrentFront = orig.CurrentFront
			}
			if val.Train.Path == nil {
				val.Train.Path = orig.Path
			}
			if val.Train.State == 0 {
				val.Train.State = orig.State
			}
			g.trains[val.TrainI] = val.Train
			g.wakeup(val.TrainI)
		case conn.ValCurrent:
			g.handleValCurrent(diffuse, val)
		case conn.ValShortNotify:
			c := g.conf.actorsReverse[diffuse.Origin]
			li := -1
			for li_, l := range g.y.Lines {
				if l.SwitchConn == (LineID{Conn: c, Line: val.Line}) {
					li = li_
				}
			}
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
			g.wakeup(ls.TakenBy)
		}
		g.publishSnapshot()
	}
}

func reverse[S ~[]E, E any](s S) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

func (g *guide) reify(ti int, t *Train) {
	log.Printf("REIFY: %s", t)
	power := t.Power
	stop := false
	max := t.CurrentFront
	if t.State == TrainStateNextAvail {
		max += 1
	}
	for i := t.CurrentBack; i <= max; i++ {
		if g.lineStates[t.Path[i].LineI].SwitchState == SwitchStateUnsafe {
			log.Printf("=== STOP UNSAFE")
			stop = true
			power = idlePower
			break
		}
	}
	stop = stop || (t.State == TrainStateNextLocked)
	if stop {
		power = idlePower
	}
	t.noPowerSupplied = power == 0
	for i := t.CurrentBack; i <= t.CurrentFront; i++ {
		g.applySwitch(ti, t, i)
		g.apply(t, i, power)
	}
	if t.State == TrainStateNextAvail {
		g.apply(t, t.next(), power)
	}
}

func (g *guide) applySwitch(ti int, t *Train, pathI int) {
	li := t.Path[pathI].LineI
	pi := t.Path[pathI].PortI
	if pi == 0 {
		// merging, so no applySwitch needed
		log.Printf("merging")
		return
	}
	// for debugging
	//if g.y.Lines[li].Comment == "W" {
	//	return
	//}
	//log.Printf("line %s", g.y.Lines[li])
	// for debugging ends here
	if g.y.Lines[li].SwitchConn == (LineID{}) {
		// no switch here
		return
	}
	if pi == 1 && g.lineStates[li].SwitchState == SwitchStateB {
		return
	} else if pi == 2 && g.lineStates[li].SwitchState == SwitchStateC {
		return
	}
	if g.lineStates[li].SwitchState == SwitchStateUnsafe {
		// already switching
		return
	}

	log.Printf("applySwitch")
	g.lineStates[li].SwitchState = SwitchStateUnsafe
	if pi == 1 {
		g.lineStates[li].nextSwitchState = SwitchStateB
	} else if pi == 2 {
		g.lineStates[li].nextSwitchState = SwitchStateC
	} else {
		panic(fmt.Sprintf("invalid pi %d", pi))
	}
	// TODO: rmbr to turn off the line afterwards!
	d := Diffuse1{
		Origin: g.conf.Actors[g.y.Lines[li].SwitchConn],
		Value: conn.ReqSwitch{
			Line:      g.y.Lines[li].SwitchConn.Line,
			Direction: pi == 1,
			Power:     180,
			Duration:  1000,
		},
	}
	//log.Printf("diffuse %#v", d)
	g.actor.OutputCh <- d
}

func (g *guide) apply(t *Train, pathI int, power int) {
	pi := t.Path[pathI].PortI
	li := t.Path[pathI].LineI
	l := g.y.Lines[li]
	rl := conn.ReqLine{
		Line: l.PowerConn.Line,
		// NOTE: reversed for now as the layout is reversed (bodge)
		// false if port A, true if port B or C
		Power: conn.AbsClampPower(power),
	}
	if pi == -1 {
		// -1 means that this LinePort is the end. Select the opposite of entryP, the port the train enters the end Line.
		prevLP := t.Path[pathI-1]
		entryP := g.y.Lines[prevLP.LineI].GetPort(prevLP.PortI).ConnP
		rl.Direction = l.GetPort(entryP).Direction
	} else {
		rl.Direction = l.GetPort(pi).Direction
	}
	// TODO: fix direction to follow layout.Layout rules
	//log.Printf("apply %s %s to %s", t, rl, g.conf.Actors[l.PowerConn])
	g.actor.OutputCh <- Diffuse1{
		Origin: g.conf.Actors[l.PowerConn],
		Value:  rl,
	}
	//log.Printf("apply2 %s", rl)
}

// syncLocks verifies locking of all currents and next (if next is available) of a train.
func (g *guide) syncLocks(ti int) {
	t := g.trains[ti]
	defer func() { g.trains[ti] = t }()
	for i := t.CurrentBack; i <= t.CurrentFront; i++ {
		ok := g.lock(t.Path[i].LineI, ti)
		if !ok {
			panic(fmt.Sprintf("train %s currents %d: locking failed", &t, i))
		}
	}
	if t.CurrentFront == len(t.Path)-1 {
		// end of path
		t.State = TrainStateNextLocked
	} else {
		ok := g.lock(t.Path[t.nextUnsafe()].LineI, ti)
		if ok {
			t.State = TrainStateNextAvail
		} else {
			t.State = TrainStateNextLocked
			log.Printf("train %d: failed to lock %d", ti, t.nextUnsafe())
		}
	}
}

func (g *guide) lock(li, ti int) (ok bool) {
	if g.lineStates[li].Taken {
		if g.lineStates[li].TakenBy != ti {
			return false
		} else {
			return true
		}
	}
	log.Printf("LOCK %d(%s) by %d", li, g.y.Lines[li].Comment, ti)
	g.lineStates[li].Taken = true
	g.lineStates[li].TakenBy = ti
	return true
}

func (g *guide) unlock(li int) {
	log.Printf("UNLOCK %d(%s) by %d", li, g.y.Lines[li].Comment, g.lineStates[li].TakenBy)
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
	Trains []Train
	Layout *layout.Layout
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
	gs := GuideSnapshot{Trains: g.trains, Layout: g.conf.Layout}
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
