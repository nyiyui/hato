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
	trainStateNextAvail  trainState = 1
	trainStateNextLocked trainState = 2
	trainStateGoal       trainState = 3
)

type train struct {
	// static fields
	power int

	// dynamic fields
	currents []int
	goal     int
	path     []LinePort
	state    trainState
}

func (t *train) String() string {
	b := new(strings.Builder)
	fmt.Fprintf(b, "power%d cur%#v", t.power, t.currents)
	switch t.state {
	case trainStateNextAvail:
		fmt.Fprintf(b, " next%d", t.path[0].LineI)
	case trainStateNextLocked:
		fmt.Fprintf(b, " locked")
	case trainStateGoal:
		fmt.Fprintf(b, " goal")
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
		power:    30,
		currents: []int{conf.Layout.MustLookupIndex("4/A")},
		goal:     conf.Layout.MustLookupIndex("1/D"),
		state:    trainStateNextAvail,
	}
	t1.path = g.y.PathTo(t1.currents[0], t1.goal)
	g.trains = append(g.trains, t1)

	go g.loop()
	return a
}

func (g *guide) loop() {
	for {
		g.single()
	}
}

// patchCurrents fixes currents that don't make sense.
// For example, a train on lines A and C but not B, which is between the two, would obviously be on B (maybe the ammeter had a problem).
func (g *guide) patchCurrents(t *train) {
	currents := make([]int, len(t.currents))
	for ci, cur := range t.currents {
		currents[ci] = cur
	}
	for ci, cur := range currents {
		if ci == len(currents)-1 {
			continue
		}
		dest := currents[ci+1]
		ok := false
		for i := 0; i < 2; i++ {
			p := g.y.Lines[cur].GetPort(i)
			if !p.ConnFilled {
				continue
			}
			if p.ConnI == dest {
				ok = true
			}
		}
		if !ok {
			lps := g.y.PathTo(cur, dest)
			missing := make([]int, 0, len(lps))
			for _, lp := range lps {
				missing = append(missing, lp.LineI)
			}
			log.Printf("patched to add %#v", missing)
			currents = append(append(currents[:ci+1], missing...), currents[ci+1:]...)
		}
		// TODO: test this
	}
	t.currents = currents
}

func (g *guide) single() {
	for ti, t := range g.trains {
		g.ensureLock(ti)
		g.reify(&t)
	}
	g.render()
	for diffuse := range g.actor.InputCh {
		var ci conn.Id
		for _, ls := range g.lineStates {
			if ls.PowerActor == diffuse.Origin {
				ci = g.conf.actorsReverse[diffuse.Origin]
			}
		}
		if ci == (conn.Id{}) {
			log.Printf("unknown conn for actor %s", diffuse.Origin)
			return
		}

		cur := diffuse.Value.(conn.ValCurrent)
		for ti, t := range g.trains {
			{
				removeCurrents := make([]int, 0, len(t.currents))
				for _, inner := range cur.Values {
					for _, li := range t.currents {
						l := g.y.Lines[li]
						if ci == l.PowerConn.Conn && inner.Line == l.PowerConn.Line && !inner.Flow {
							removeCurrents = append(removeCurrents, li)
						}
					}
				}
				for _, li := range removeCurrents {
					g.unlock(li)
					g.apply(li, 0)
				}
			}
			{
				keepCurrents := make([]int, 0, len(t.currents))
				for _, inner := range cur.Values {
					for _, li := range t.currents {
						l := g.y.Lines[li]
						if ci == l.PowerConn.Conn && inner.Line == l.PowerConn.Line && inner.Flow {
							keepCurrents = append(keepCurrents, li)
						}
					}
				}
				t.currents = keepCurrents
			}
			{
				for _, li := range t.currents {
					if li == t.goal {
						t.state = trainStateGoal
						break
					}
				}
			}
			if t.state == trainStateNextAvail {
				for _, inner := range cur.Values {
					nextI := t.path[0].LineI
					next := g.y.Lines[nextI]
					if ci == next.PowerConn.Conn && inner.Line == next.PowerConn.Line && inner.Flow {
						t.currents = append(t.currents, nextI)
						t.path = t.path[1:]
						g.tryLockingNext(ti, &t)
						break
					}
				}
			}
			g.patchCurrents(&t)
			g.trains[ti] = t
			log.Printf("train: postshow: %s", &t)
		}
		g.render()
		for ti, t := range g.trains {
			g.tryLockingNext(ti, &t)
			g.reify(&t)
			g.trains[ti] = t
		}
		g.render()
	}
}

func (g *guide) tryLockingNext(ti int, t *train) {
	// try locking next again
	nextI := t.path[0].LineI
	ok := g.lock(nextI, ti)
	if ok {
		t.state = trainStateNextAvail
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
	switch t.state {
	case trainStateNextAvail:
		for _, li := range t.currents {
			g.apply(li, t.power)
		}
		g.apply(t.path[0].LineI, t.power)
	case trainStateNextLocked, trainStateGoal:
		for _, li := range t.currents {
			g.apply(li, idlePower)
		}
	}
	log.Printf("REIFY DONE: %s", t)
}

func (g *guide) apply(li int, power int) {
	rl := conn.ReqLine{
		Line:      g.y.Lines[li].PowerConn.Line,
		Direction: power > 0,
		Power:     conn.AbsClampPower(power),
	}
	// TODO: fix direction to follow layout.Layout rules
	// log.Printf("apply %s %s", li, rl)
	g.actor.OutputCh <- Diffuse1{
		Origin: g.conf.Actors[g.y.Lines[li].PowerConn],
		Value:  rl,
	}
	//log.Printf("apply2 %s", rl)
}

// ensureLock verifies locking of all currents and next (if next is available) of a train.
func (g *guide) ensureLock(ti int) {
	t := g.trains[ti]
	for ci, cur := range t.currents {
		ok := g.lock(cur, ti)
		if !ok {
			panic(fmt.Sprintf("train %s currents %d: locking failed", &t, ci))
		}
	}
	if t.state == trainStateNextAvail {
		ok := g.lock(t.path[0].LineI, ti)
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
	log.Printf("LOCK %d(%s) by %d", li, g.y.Lines[li].Comment, ti)
	g.lineStates[li].Taken = true
	g.lineStates[li].TakenBy = ti
	return true
}

func (g *guide) unlock(li int) {
	log.Printf("UNLOCK %d(%s) by %d", li, g.y.Lines[li].Comment, g.lineStates[li].TakenBy)
	g.lineStates[li].Taken = false
	g.lineStates[li].TakenBy = -1
}
