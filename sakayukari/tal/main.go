package tal

import (
	"errors"
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

const idlePower = 10

// guide - uses line to move trains
// adjuster - adjusts power level etc

type GuideConf struct {
	Layout *layout.Layout
	Actors map[LineID]ActorRef
}

type train struct {
	// static fields
	power int

	// dynamic fields
	currents  []int
	next      int
	nextAvail bool
}

func (t *train) String() string {
	b := new(strings.Builder)
	fmt.Fprintf(b, "power%d cur%s", t.power, t.currents)
	if t.nextAvail {
		fmt.Fprintf(b, " next%s", t.next)
	} else {
		fmt.Fprintf(b, " next-NA")
	}
	return b.String()
}

type guide struct {
	actor      Actor
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
		fmt.Fprintf(b, "%d %s", ti, t)
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
	g := guide{
		actor:      a,
		trains:     make([]train, 0),
		lineStates: make([]lineState, len(conf.Layout.Lines)),
		y:          conf.Layout,
		state:      widgets.NewParagraph(),
	}
	g.state.SetRect(0, 6, 30, 16)
	g.trains = append(g.trains, train{
		power:     30,
		currents:  []int{conf.Layout.MustLookupIndex("5")},
		next:      conf.Layout.MustLookupIndex("4"),
		nextAvail: true,
	})
	go g.loop()
	return a
}

func (g *guide) loop() {
	for {
		g.single()
	}
}

func (g *guide) next(t train, li LineID) (li2 LineID, exists bool, err error) {
	list := []string{"A", "B", "C", "D"}
	i := li.Line[0] - 'A'
	if t.power == 0 {
		return LineID{}, false, errors.New("no power")
	}
	if t.power > 0 {
		i++
	}
	if t.power < 0 {
		i--
	}
	if i < 0 || i > 3 {
		return LineID{}, false, nil
	}
	return LineID{li.Conn, list[i]}, true, nil
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
		next, exists := g.y.Step(cur, t.power > 0)
		if !exists {
			panic(fmt.Sprintf("next of %s's %d index current (current = %s) is nonexistent", t, ci, cur))
		}
		_ = next
		if next != currents[ci+1] {
			// missing a line in between
			log.Printf("patched to add %s", next)
			currents = append(append(currents[:ci+1], next), currents[ci+1:]...)
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
				ci = ls.ID.Conn
			}
		}
		if ci == (conn.Id{}) {
			log.Printf("unknown conn for actor %s", diffuse.Origin)
			return
		}

		cur := diffuse.Value.(conn.ValCurrent)
		for ti, t := range g.trains {
			{
				removeCurrents := make([]LineID, 0, len(t.currents))
				for _, inner := range cur.Values {
					for _, li := range t.currents {
						l := g.y.Lines[li]
						if ci == l.PowerConn.Conn && inner.Line == l.PowerConn.Line && !inner.Flow {
							removeCurrents = append(removeCurrents, cur)
						}
					}
				}
				for _, cur := range removeCurrents {
					g.unlock(cur)
					g.apply(cur, 0)
				}
			}
			{
				keepCurrents := make([]LineID, 0, len(t.currents))
				for _, inner := range cur.Values {
					for _, li := range t.currents {
						l := g.y.Lines[li]
						if ci == l.PowerConn.Conn && inner.Line == l.PowerConn.Line && inner.Flow {
							keepCurrents = append(keepCurrents, cur)
						}
					}
				}
				t.currents = keepCurrents
			}
			if t.nextAvail {
			NextLoop:
				for _, inner := range cur.Values {
					if inner.Line == t.next.Line && inner.Flow {
						t.currents = append(t.currents, t.next)
						err := g.recalcNextAndLock(ti, &t)
						if err != nil {
							log.Printf("train %d: %s", ti, err)
							panic("not implemented yet")
						}
						break NextLoop
					}
				}
			}
			g.patchCurrents(&t)
			g.trains[ti] = t
			log.Printf("train: postshow: %s", &t)
		}
		g.render()
		for ti, t := range g.trains {
			g.recalcNextAndLock(ti, &t)
			g.reify(&t)
			g.trains[ti] = t
		}
		g.render()
	}
}

func reverse[S ~[]E, E any](s S) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

func (g *guide) recalcNextAndLock(ti int, t *train) error {
	err := g.recalcNext(t)
	if err != nil {
		return err
	}
	log.Printf("recalcNextAndLock %d: next: %s", ti, t.next)
	if t.nextAvail {
		ok := g.lock(t.next, ti)
		if !ok {
			t.nextAvail = false
		}
		log.Printf("recalcNextAndLock %d: lock: %t", ti, ok)
	}
	return nil
}

func (g *guide) recalcNext(t *train) error {
	if len(t.currents) == 0 {
		return errors.New("no currents")
	}
	newNext, exists, err := g.next(*t, t.currents[len(t.currents)-1])
	if err != nil {
		return err
	}
	if exists {
		t.next = newNext
		t.nextAvail = true
	} else {
		t.next = -1
		t.nextAvail = false
	}
	return nil
}

func (g *guide) reify(t *train) {
	log.Printf("REIFY: %s", t)
	if t.nextAvail {
		for _, li := range t.currents {
			g.apply(li, t.power)
		}
		g.apply(t.next, t.power)
	} else {
		for _, li := range t.currents {
			g.apply(li, idlePower)
		}
	}
	log.Printf("REIFY DONE: %s", t)
}

func (g *guide) apply(li int, power int) {
	rl := conn.ReqLine{
		Line:      li.Line,
		Direction: power > 0,
		Power:     conn.AbsClampPower(power),
	}
	// TODO: fix direction to follow layout.Layout rules
	// log.Printf("apply %s %s", li, rl)
	g.actor.OutputCh <- Diffuse1{
		Origin: g.y.Lines[li].PowerConn,
		Value:  rl,
	}
	//log.Printf("apply2 %s", rl)
}

// ensureLock verifies locking of all currents and next (if nextAvail is true) of a train.
func (g *guide) ensureLock(ti int) {
	t := g.trains[ti]
	for ci, cur := range t.currents {
		ok := g.lock(cur, ti)
		if !ok {
			panic(fmt.Sprintf("train %s currents %d: locking failed", &t, ci))
		}
	}
	if t.nextAvail {
		ok := g.lock(t.next, ti)
		if !ok {
			panic(fmt.Sprintf("train %s netx: locking failed", &t))
		}
	}
	g.trains[ti] = t
}

func (g *guide) lock(li, ti int) (ok bool) {
	if g.lineStates[i].Taken && g.lineStates[i].TakenBy != ti {
		return false
	}
	log.Printf("LOCK %s", li)
	g.lineStates[i].Taken = true
	g.lineStates[i].TakenBy = ti
	return true
}

func (g *guide) unlock(li int) {
	log.Printf("UNLOCK %s", li)
	g.lineStates[li].Taken = false
}
