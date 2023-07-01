package tal

import (
	"errors"
	"fmt"
	"log"

	. "nyiyui.ca/hato/sakayukari"
	"nyiyui.ca/hato/sakayukari/conn"
)

// guide - uses line to move trains
// adjuster - adjusts power level etc

type GuideConf struct {
	Lines []LineConf
}

type LineConf struct {
	Actor ActorRef
	Conn  conn.Id
}

type LineID struct {
	Conn conn.Id
	// Usually A, B, C, or D.
	Line string
}

func (li LineID) String() string {
	return fmt.Sprintf("%s::%s", li.Conn, li.Line)
}

type train struct {
	// static fields
	power int

	// dynamic fields
	current LineID
	next    LineID
}

func (t *train) String() string {
	return fmt.Sprintf("power%d cur%s next%s", t.power, t.current, t.next)
}

type line struct {
	Actor   ActorRef
	ID      LineID
	TakenBy int
}

type guide struct {
	actor  Actor
	trains []train
	lines  []line
}

func Guide(conf GuideConf) Actor {
	a := Actor{
		Comment:  "tal-guide",
		InputCh:  make(chan Diffuse1),
		OutputCh: make(chan Diffuse1),
		Inputs:   make([]ActorRef, 0, len(conf.Lines)),
		Type: ActorType{
			Input:       true,
			LinearInput: true,
			Output:      true,
		},
	}
	for _, lc := range conf.Lines {
		a.Inputs = append(a.Inputs, lc.Actor)
	}
	g := guide{
		actor:  a,
		trains: make([]train, 0),
		lines:  make([]line, 0),
	}
	g.trains = append(g.trains, train{
		power:   40,
		current: LineID{Conn: conf.Lines[0].Conn, Line: "A"},
		next:    LineID{Conn: conf.Lines[0].Conn, Line: "B"},
	})
	for _, lc := range conf.Lines {
		lines := []string{"A", "B", "C", "D"}
		for _, l := range lines {
			g.lines = append(g.lines, line{
				Actor: lc.Actor,
				ID: LineID{
					Conn: lc.Conn,
					Line: l,
				},
				TakenBy: -1,
			})
		}
	}
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

func (g *guide) single() {
	for ti, t := range g.trains {
		log.Printf("train %d: %s", ti, &t)
		g.apply(t.current, t.power)
		g.apply(t.next, t.power)
	}
	for diffuse := range g.actor.InputCh {
		log.Print("new diffuse")
		var ci conn.Id
		for _, l := range g.lines {
			if l.Actor == diffuse.Origin {
				ci = l.ID.Conn
			}
		}
		if ci == (conn.Id{}) {
			log.Printf("unknown conn for actor %s", diffuse.Origin)
			return
		}
		cur := diffuse.Value.(conn.ValCurrent)
		for ti, t := range g.trains {
			if t.next.Conn != ci {
				log.Print("diff conn")
				continue
			}
			for _, inner := range cur.Values {
				if inner.Line == t.next.Line && inner.Flow {
					log.Printf("train %d: next")
					g.unlock(t.current)
					g.lock(t.next, ti)
					newCurrent, exists, err := g.next(t, t.current)
					if err != nil {
						log.Printf("train %d: %s", ti, err)
						panic("not implemented yet")
					}
					if !exists {
						panic("what‽")
					}
					newNext, exists, err := g.next(t, t.next)
					if err != nil {
						log.Printf("train %d: %s", ti, err)
						panic("not implemented yet")
					}
					if !exists {
						t.power = 0
					}
					t.current = newCurrent
					t.next = newNext
					g.apply(t.current, t.power)
					g.apply(t.next, t.power)
				}
				// TODO: locking mechanism (not for now because we only have 1 train)
			}
		}
	}
}

// Returns -1 if nonexistent.
func (g *guide) findLine(li LineID) int {
	for i, l := range g.lines {
		if l.ID == li {
			return i
		}
	}
	return -1
}

func (g *guide) lock(li LineID, ti int) {
	g.lines[g.findLine(li)].TakenBy = ti
}

func (g *guide) unlock(li LineID) {
	g.lines[g.findLine(li)].TakenBy = -1
}

func (g *guide) apply(li LineID, power int) {
	rl := conn.ReqLine{
		Line:      li.Line,
		Direction: power > 0,
		Power:     conn.AbsClampPower(power),
	}
	//log.Printf("apply %s", rl)
	g.actor.OutputCh <- Diffuse1{
		Origin: g.lines[g.findLine(li)].Actor,
		Value:  rl,
	}
	//log.Printf("apply2 %s", rl)
}
