package tal

import (
	"fmt"
	"log"
	"strings"

	"github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	. "nyiyui.ca/hato/sakayukari"
	"nyiyui.ca/hato/sakayukari/conn"
	"nyiyui.ca/hato/sakayukari/tal/layout"
)

type LineID = layout.LineID
type LinePort = layout.LinePort

const idlePower = 10

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
	// trainStateGoal means the train is at the goal. The train should stop.
	trainStateGoal trainState = 3
)

type train struct {
	// power supplied directly to soyuu-line.
	power int

	// dynamic fields

	// currentBack is the path index of the last car's occupying line.
	currentBack int
	// currentFront is the path index of the first car's occupying line.
	currentFront int
	// next is the path index of the first car's net line.
	next int
	// path is the path of outgoing LinePorts until the goal.
	path  []LinePort
	state trainState
}

func (t *train) String() string {
	b := new(strings.Builder)
	fmt.Fprintf(b, "power%d %d-%d", t.power, t.currentBack, t.currentFront)
	switch t.state {
	case trainStateNextAvail:
		fmt.Fprintf(b, "→%d", t.next)
	case trainStateNextLocked:
		fmt.Fprintf(b, "L")
	case trainStateGoal:
		fmt.Fprintf(b, "G")
	}
	for _, lp := range t.path {
		fmt.Fprintf(b, " %s", lp)
	}
	return b.String()
}

type guide struct {
	actor      Actor
	conf       GuideConf
	trains     []train
	lineStates []lineState
	y          *layout.Layout
	state      *widgets.Paragraph
}

type lineState struct {
	Taken       bool
	TakenBy     int
	PowerActor  ActorRef
	SwitchActor ActorRef
}

func (g *guide) render() {
	b := new(strings.Builder)
	for ti, t := range g.trains {
		fmt.Fprintf(b, "%d %#v", ti, t)
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
		trains:     make([]train, 0),
		lineStates: make([]lineState, len(conf.Layout.Lines)),
		y:          conf.Layout,
		state:      widgets.NewParagraph(),
	}
	g.state.SetRect(0, 6, 70, 20)
	t1 := train{
		power:        40,
		currentBack:  0,
		currentFront: 0,
		next:         1,
		state:        trainStateNextAvail,
	}
	t1.path = g.y.PathTo(conf.Layout.MustLookupIndex("4/A"), conf.Layout.MustLookupIndex("1/D"))
	g.trains = append(g.trains, t1)

	go g.loop()
	return a
}

func (g *guide) loop() {
	for ti, t := range g.trains {
		g.ensureLock(ti)
		g.reify(&t)
	}
	g.render()
	for diffuse := range g.actor.InputCh {
		ci, ok := g.conf.actorsReverse[diffuse.Origin]
		if !ok {
			log.Printf("unknown conn for actor %s", diffuse.Origin)
			return
		}

		cur := diffuse.Value.(conn.ValCurrent)
		log.Printf("=== diffuse from %s: %s", ci, cur)
		for ti, t := range g.trains {
			for _, inner := range cur.Values {
				cd := g.y.Lines[t.path[t.currentBack].LineI]
				if ci == cd.PowerConn.Conn && inner.Line == cd.PowerConn.Line && !inner.Flow {
					nextI := t.path[t.currentBack].LineI
					g.unlock(nextI)
					g.apply(&t, t.currentBack, 0)
					t.currentBack++
					log.Printf("=== new currentBack: %d", t.currentBack)
					continue
				}
				cf := g.y.Lines[t.path[t.next].LineI]
				// if t.state ≠ trainStateNextAvail, the reaction could be the train in front of this train
				if t.state == trainStateNextAvail && ci == cf.PowerConn.Conn && inner.Line == cf.PowerConn.Line && inner.Flow {
					log.Printf("=== next activated: %d", t.next)
					t.currentFront++
					t.next++
					g.tryLockingNext(ti, &t)
					continue
				}
			}
			// TODO: check if the train derailed, was removed, etc (come up with a heuristic)
			// TODO: check for overruns
			if t.currentFront == len(t.path)-1 {
				t.state = trainStateGoal
			}
			g.trains[ti] = t
			log.Printf("postshow: %s", &t)
		}
		g.render()
		for ti, t := range g.trains {
			log.Printf("postshow2: %s", &t)
			g.tryLockingNext(ti, &t)
			g.reify(&t)
			g.trains[ti] = t
		}
		g.render()
	}
}

func (g *guide) tryLockingNext(ti int, t *train) {
	nextI := t.path[t.next].LineI
	ok := g.lock(nextI, ti)
	if ok {
		t.state = trainStateNextAvail
		log.Printf("train %d: successfully locked %d", ti, nextI)
	} else {
		t.state = trainStateNextLocked
		log.Printf("train %d: failed to lock %d", ti, nextI)
	}
	g.ensureLock(ti)
}

func reverse[S ~[]E, E any](s S) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

func (g *guide) reify(t *train) {
	// TODO: handle switches
	log.Printf("REIFY: %s", t)
	power := 0
	switch t.state {
	case trainStateNextAvail:
		power = t.power
		g.apply(t, t.next, power)
	case trainStateNextLocked, trainStateGoal:
		power = idlePower
	}
	for i := t.currentBack; i <= t.currentFront; i++ {
		log.Printf("REIFY STEP: %d", i)
		g.apply(t, i, power)
	}
	log.Printf("REIFY DONE: %s", t)
}

func (g *guide) apply(t *train, pathI int, power int) {
	li := t.path[pathI].LineI
	rl := conn.ReqLine{
		Line: g.y.Lines[li].PowerConn.Line,
		//Direction: t.path[pathI].PortI != 0,
		Direction: t.path[pathI].PortI == 0,
		// NOTE: reversed for now as the layout is reversed (bodge)
		// false if port A, true if port B or C
		Power: conn.AbsClampPower(power),
	}
	// TODO: fix direction to follow layout.Layout rules
	log.Printf("apply %s %s", t, rl)
	log.Printf("g.actor.OutputCh %#v", g.actor.OutputCh)
	g.actor.OutputCh <- Diffuse1{
		Origin: g.conf.Actors[g.y.Lines[li].PowerConn],
		Value:  rl,
	}
	log.Printf("apply2 %s", rl)
}

// ensureLock verifies locking of all currents and next (if next is available) of a train.
func (g *guide) ensureLock(ti int) {
	t := g.trains[ti]
	for i := t.currentBack; i <= t.currentFront; i++ {
		ok := g.lock(t.path[i].LineI, ti)
		if !ok {
			panic(fmt.Sprintf("train %s currents %d: locking failed", &t, i))
		}
	}
	if t.state == trainStateNextAvail {
		ok := g.lock(t.path[t.next].LineI, ti)
		if !ok {
			panic(fmt.Sprintf("train %s netx: locking failed", &t))
		}
	}
	g.trains[ti] = t
}

func (g *guide) lock(li, ti int) (ok bool) {
	if g.lineStates[li].Taken && g.lineStates[li].TakenBy != ti {
		return false
	}
	if g.lineStates[li].TakenBy != ti {
		return true
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
	// TODO: maybe do tryLockingNext for all trains that match (instead of the dumb for loop in guide.single())
}
