package tal

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
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

type trainState int

const (
	// trainStateNextAvail means the next line is available. The train should move to the next line.
	trainStateNextAvail trainState = 1
	// trainStateNextLocked means the next line is locked by another train. The train should stop and wait at its current position, unless a precise attitude is available. If a precise attitude is available, it should stop without entering the next line.
	trainStateNextLocked trainState = 2
)

type Train struct {
	// power supplied directly to soyuu-line (when moving)
	power           int
	noPowerSupplied bool

	// dynamic fields

	// currentBack is the path index of the last car's occupying line.
	currentBack int
	// currentFront is the path index of the first car's occupying line.
	currentFront int
	// path is the path of outgoing LinePorts until the goal.
	path  []LinePort
	state trainState
}

// nextUnsafe returns the path index of the next LinePort.
// Note: this does check if this train has a next available, and panics if next is not available.
func (t *Train) next() int {
	if t.state != trainStateNextAvail {
		panic("next() called when not trainStateNextAvail")
	}
	return t.nextUnsafe()
}

// nextUnsafe returns the path index of the next LinePort.
// Note: this does not check if this train has a next available.
func (t *Train) nextUnsafe() int {
	return t.currentFront + 1
}

func (t *Train) String() string {
	b := new(strings.Builder)
	fmt.Fprintf(b, "power%d %d-%d", t.power, t.currentBack, t.currentFront)
	switch t.state {
	case trainStateNextAvail:
		fmt.Fprintf(b, "→%d", t.next())
	case trainStateNextLocked:
		fmt.Fprintf(b, "L")
	}
	for _, lp := range t.path {
		fmt.Fprintf(b, " %s", lp)
	}
	return b.String()
}

type GuideSnapshot struct {
	Trains []Train
}

type guide struct {
	actor      Actor
	conf       GuideConf
	trains     []Train
	lineStates []lineState
	y          *layout.Layout
	state      *widgets.Paragraph
}

type lineState struct {
	Taken           bool
	TakenBy         int
	PowerActor      ActorRef
	SwitchActor     ActorRef
	SwitchState     SwitchState
	nextSwitchState SwitchState
}

func (g *guide) render() {
	b := new(strings.Builder)
	for ti, t := range g.trains {
		fmt.Fprintf(b, "%d %s\n", ti, &t)
		fmt.Fprintf(b, "%d %#v\n", ti, t)
	}
	g.state.Text = b.String()
	termui.Render(g.state)
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
		state:      widgets.NewParagraph(),
	}
	g.state.SetRect(0, 6, 70, 20)
	t1 := Train{
		power:        70,
		currentBack:  0,
		currentFront: 0,
		state:        trainStateNextAvail,
	}
	//t1.path = g.y.PathTo(g.y.MustLookupIndex("Y"), g.y.MustLookupIndex("W")) // reverse
	t1.path = g.y.PathTo(g.y.MustLookupIndex("Y"), g.y.MustLookupIndex("X")) // normal
	{
		last := t1.path[len(t1.path)-1]
		p := g.y.Lines[last.LineI].GetPort(last.PortI)
		t1.path = append(t1.path, LinePort{LineI: p.ConnI, PortI: -1})
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

			cb := g.y.Lines[t.path[t.currentBack].LineI]
			if ci == cb.PowerConn.Conn && inner.Line == cb.PowerConn.Line && !inner.Flow {
				if t.currentBack >= t.currentFront {
					// this can happen e.g. when the train is at 0-0→1 and then the 0th line becomes 0 (e.g. A0, B0)
					goto NoCurrentBack
				}
				nextI := t.path[t.currentBack].LineI
				g.unlock(nextI)
				g.apply(&t, t.currentBack, 0)
				t.currentBack++
				log.Printf("=== currentBack succession: %d", t.currentBack)
			}
		NoCurrentBack:
			cf := g.y.Lines[t.path[t.currentFront].LineI]
			if ci == cf.PowerConn.Conn && inner.Line == cf.PowerConn.Line && !inner.Flow {
				if t.currentFront == 0 {
					log.Printf("=== currentFront regression (ignore): %d", t.currentFront)
					goto NoCurrentFront
				}
				if t.currentFront <= t.currentBack {
					// this can happen e.g. when the train is at 1-1→2 and then the 1st line becomes 0 (e.g. A0, B0) (currentBack moving to 0 is prevented by an if for currentBack)
					log.Printf("=== currentFront regression (ignore as currentFront <= currentBack): %d", t.currentFront)
					goto NoCurrentFront
				}
				nextI := t.path[t.currentFront].LineI
				g.unlock(nextI)
				g.apply(&t, t.currentFront, 0)
				t.currentFront--
				log.Printf("=== currentFront regression: %d", t.currentFront)
			}
		NoCurrentFront:
			if t.state == trainStateNextAvail {
				// if t.state ≠ trainStateNextAvail, t.next could be out of range
				cf := g.y.Lines[t.path[t.next()].LineI]
				if ci == cf.PowerConn.Conn && inner.Line == cf.PowerConn.Line && inner.Flow {
					t.currentFront++
					log.Printf("=== next succession: %d", t.currentFront)
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
	g.render()
	for ti := range g.trains {
		g.wakeup(ti)
		log.Printf("postwakeup: %s", &g.trains[ti])
	}
	g.render()
}

func (g *guide) wakeup(ti int) {
	g.syncLocks(ti)
	t := g.trains[ti]
	g.reify(ti, &t)
	g.trains[ti] = t
}

func (g *guide) loop() {
	time.Sleep(1 * time.Second)
	for ti := range g.trains {
		g.wakeup(ti)
	}
	g.render()
	for diffuse := range g.actor.InputCh {
		switch val := diffuse.Value.(type) {
		case conn.ValCurrent:
			g.handleValCurrent(diffuse, val)
		case conn.ValShortNotify:
			log.Printf("diffuse ValShortNotify")
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
	}
}

func reverse[S ~[]E, E any](s S) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

func (g *guide) reify(ti int, t *Train) {
	log.Printf("REIFY: %s", t)
	power := t.power
	stop := false
	max := t.currentFront
	if t.state == trainStateNextAvail {
		max += 1
	}
	for i := t.currentBack; i <= max; i++ {
		if g.lineStates[t.path[i].LineI].SwitchState == SwitchStateUnsafe {
			log.Printf("=== STOP UNSAFE")
			stop = true
			power = idlePower
			break
		}
	}
	stop = stop || (t.state == trainStateNextLocked)
	if stop {
		power = idlePower
	}
	t.noPowerSupplied = power == 0
	for i := t.currentBack; i <= t.currentFront; i++ {
		g.applySwitch(ti, t, i)
		g.apply(t, i, power)
	}
	if t.state == trainStateNextAvail {
		g.apply(t, t.next(), power)
	}
}

func (g *guide) applySwitch(ti int, t *Train, pathI int) {
	li := t.path[pathI].LineI
	pi := t.path[pathI].PortI
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
		log.Printf("applySwitch already switching")
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
	li := t.path[pathI].LineI
	line := g.y.Lines[li]
	rl := conn.ReqLine{
		Line: line.PowerConn.Line,
		//Direction: t.path[pathI].PortI != 0,
		Direction: t.path[pathI].PortI == 0,
		// NOTE: reversed for now as the layout is reversed (bodge)
		// false if port A, true if port B or C
		Power: conn.AbsClampPower(power),
	}
	// TODO: fix direction to follow layout.Layout rules
	//log.Printf("apply %s %s to %s", t, rl, g.conf.Actors[line.PowerConn])
	g.actor.OutputCh <- Diffuse1{
		Origin: g.conf.Actors[line.PowerConn],
		Value:  rl,
	}
	//log.Printf("apply2 %s", rl)
}

// syncLocks verifies locking of all currents and next (if next is available) of a train.
func (g *guide) syncLocks(ti int) {
	t := g.trains[ti]
	defer func() { g.trains[ti] = t }()
	for i := t.currentBack; i <= t.currentFront; i++ {
		ok := g.lock(t.path[i].LineI, ti)
		if !ok {
			panic(fmt.Sprintf("train %s currents %d: locking failed", &t, i))
		}
	}
	if t.currentFront == len(t.path)-1 {
		// end of path
		t.state = trainStateNextLocked
	} else {
		ok := g.lock(t.path[t.nextUnsafe()].LineI, ti)
		if ok {
			t.state = trainStateNextAvail
		} else {
			t.state = trainStateNextLocked
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
