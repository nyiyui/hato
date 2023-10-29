package tal

import (
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/exp/slices"
	. "nyiyui.ca/hato/sakayukari"
	"nyiyui.ca/hato/sakayukari/conn"
	"nyiyui.ca/hato/sakayukari/notify"
	"nyiyui.ca/hato/sakayukari/tal/layout"
)

type Simulator struct {
	Comment      string
	lineStates   []lineState
	trainStates  []trainState
	actors       []ActorAndRef
	g            *Guide
	conns        map[conn.Id]ActorRef
	connsReverse map[ActorRef]conn.Id
	events       *notify.Multiplexer[Event]
	eventsS      *notify.MultiplexerSender[Event]
}

func NewSimulator(comment string) *Simulator {
	s := &Simulator{
		Comment: comment,
	}
	s.eventsS, s.events = notify.NewMultiplexerSender[Event](comment)
	return s
}

func (s *Simulator) SetGuide(g *Guide) {
	s.g = g
}

type Event interface {
	fmt.Stringer
}

type EventEntryExit struct {
	LineI layout.LineI
	// Enter is whether this was an enter or exit event.
	Enter bool
}

func (enx EventEntryExit) String() string {
	verb := map[bool]string{true: "enter", false: "exit"}[enx.Enter]
	return fmt.Sprintf("%s line %d", verb, enx.LineI)
}

type EventPower struct {
	LineI layout.LineI
	// Power is the new power set.
	// Note that the sign is not related to Port.Direction in Layout.
	// Positive = towards port A, negative = towards B/C
	Power int
}

func (ep EventPower) String() string {
	return fmt.Sprintf("set power %d to line %d", ep.Power, ep.LineI)
}

func (s *Simulator) GenActorRefs(newActor func(a Actor) ActorRef) map[layout.LineID]ActorRef {
	res := map[layout.LineID]ActorRef{}
	s.conns = map[conn.Id]ActorRef{}
	ensureConn := func(l LineID) {
		c := l.Conn
		if c == (conn.Id{}) {
			return
		}
		ref, ok := s.conns[c]
		if !ok {
			a := Actor{
				Comment:  fmt.Sprintf("%s conn %s", s.Comment, c),
				OutputCh: make(chan Diffuse1),
				InputCh:  make(chan Diffuse1),
				Type: ActorType{
					Input:       true,
					LinearInput: true,
					Output:      true,
				},
			}
			s.conns[c] = newActor(a)
			ref = s.conns[c]
			s.actors = append(s.actors, ActorAndRef{a, ref})
		}
		res[l] = ref
	}
	for _, line := range s.g.Layout.Lines {
		ensureConn(line.PowerConn)
		ensureConn(line.SwitchConn)
	}
	s.connsReverse = reverseMap(s.conns)
	return res
}

func (s *Simulator) Run() {
	s.g.Model2.setIgnoreWrites()
	s.lineStates = make([]lineState, len(s.g.Layout.Lines))
	s.trainStates = make([]trainState, len(s.g.trains))
	go s.logEvents()
	for i := range s.actors {
		go s.handleActorOutput(i)
		go s.handleActorInput(i)
	}
	go s.simulateTrains()
}

func (s *Simulator) handleActorOutput(i int) {
	aar := s.actors[i]
	current := map[string]bool{}
	current["A"] = false
	current["B"] = false
	current["C"] = false
	current["D"] = false
	current["E"] = false
	current["F"] = false
	current["G"] = false
	current["H"] = false
	ch := make(chan Event)
	s.events.Subscribe(fmt.Sprintf("actor %d", i), ch)
	defer s.events.Unsubscribe(ch)
	for ev := range ch {
		switch ev := ev.(type) {
		case EventEntryExit:
			line := s.g.Layout.Lines[ev.LineI]
			if s.conns[line.PowerConn.Conn] != aar.Ref {
				continue
			}
			changed := current[line.PowerConn.Line] != ev.Enter
			if !changed {
				continue
			}
			current[line.PowerConn.Line] = ev.Enter
			// send to OutputCh
			vc := conn.ValCurrent{Values: make([]conn.ValCurrentInner, 0, len(current))}
			for domain, flow := range current {
				vc.Values = append(vc.Values, conn.ValCurrentInner{
					Line: domain,
					Flow: flow,
				})
			}
			vc.Sort()
			zap.S().Infof("send: %s", Diffuse1{Value: vc})
			aar.Actor.OutputCh <- Diffuse1{Value: vc}
		default:
			continue
		}
	}
}

func (s *Simulator) handleActorInput(i int) {
	aar := s.actors[i]
	for diffuse := range aar.Actor.InputCh {
		connId, ok := s.connsReverse[aar.Ref]
		if !ok {
			panic("unreachable")
		}
		zap.S().Infof("recv: %s", diffuse)
		switch val := diffuse.Value.(type) {
		case conn.ReqSwitch:
			li := LineID{connId, val.Line}
			lineI := slices.IndexFunc(s.g.Layout.Lines, func(l layout.Line) bool {
				return l.PowerConn == li || l.SwitchConn == li
			})
			line := s.g.Layout.Lines[lineI]
			if line.PowerConn == li {
				panic("ReqSwitch of PowerConn not supported")
			}
			func() {
				s.lineStates[lineI].Lock.Lock()
				defer s.lineStates[lineI].Lock.Unlock()
				s.lineStates[lineI].SwitchState = SwitchStateUnsafe
			}()
			goalSwitchState := map[bool]SwitchState{true: SwitchStateB, false: SwitchStateC}[val.Direction]
			timer := time.NewTimer(time.Duration(val.Duration) * time.Millisecond)
			go func() {
				<-timer.C
				s.lineStates[lineI].Lock.Lock()
				defer s.lineStates[lineI].Lock.Unlock()
				s.lineStates[lineI].SwitchState = goalSwitchState
				diffuse := Diffuse1{
					Value: conn.ValShortNotify{
						Line:      val.Line,
						Monotonic: time.Now().UnixMilli(), // not monotonic but should be good enough (I don't think it's used anyways)
					},
				}
				zap.S().Infof("send: %s", diffuse)
				aar.Actor.OutputCh <- diffuse
			}()
		case conn.ReqLine:
			li := LineID{connId, val.Line}
			lineI := slices.IndexFunc(s.g.Layout.Lines, func(l layout.Line) bool {
				return l.PowerConn == li || l.SwitchConn == li
			})
			line := s.g.Layout.Lines[lineI]
			if line.SwitchConn == li {
				panic("ReqLine of SwitchConn not supported")
			}
			powerCoeff := +1
			if line.PortB.Direction == val.Direction || line.PortC.Direction == val.Direction {
				powerCoeff = -1
			}
			s.eventsS.Send(EventPower{
				LineI: layout.LineI(lineI),
				Power: powerCoeff * int(val.Power),
			})
		default:
			continue
		}
	}
}

type trainState struct {
	Position Position
	Train    Train
}

func (s *Simulator) simulateTrains() {
	snap := s.g.snapshot()
	for i := range s.g.trains {
		s.trainStates[i].Train = snap.Trains[i]
	}
	stepI := 0
	for range time.NewTicker(100 * time.Millisecond).C {
		s.step(stepI)
		stepI++
	}
}

func (s *Simulator) step(stepI int) {
	// kinda racy but /shrug... most guide calls should happen at the end of this step (after which there is a pause)
	for i := range s.g.trains {
		ts := s.trainStates[i]
		oldT := ts.Train
		newT := s.g.trains[i]
		s.checkTrainPower(i)
		form := s.g.conf.Cars.Forms[oldT.FormI]
		prevPos := ts.Position
		prevPos2 := s.g.Layout.MustOffsetToPosition(*oldT.Path, abs(s.g.Layout.PositionToOffset(*oldT.Path, ts.Position)-int64(form.Length)))
		newPos, overrun := s.g.Model2.CurrentPosition3(&newT, false)
		if overrun {
			zap.S().Errorf("step %d: train %d overran (front)", stepI, i)
		}
		newPos2 := s.g.Layout.MustOffsetToPosition(*newT.Path, abs(s.g.Layout.PositionToOffset(*newT.Path, newPos)-int64(form.Length)))
		zap.S().Infof("step %d: train %d: power = %d ; position = %s → %s ; path = %s", stepI, i, newT.Power, newPos2, newPos, *newT.Path)
		// TODO: check if t.Path changed (we don't support changing t.Path)
		// give up if we moved more than 1 line away (code is too complex D:)
		events := make([]Event, 0)
		prevPosI := slices.IndexFunc(newT.Path.Follows, func(lp layout.LinePort) bool {
			return lp.LineI == prevPos.LineI
		})
		for i := prevPosI + 1; i < len(newT.Path.Follows) && newT.Path.Follows[i].LineI == newPos.LineI; i++ {
			enx := EventEntryExit{
				LineI: newT.Path.Follows[i].LineI,
				Enter: true,
			}
			events = append(events, enx)
		}
		prevPos2I := slices.IndexFunc(newT.Path.Follows, func(lp layout.LinePort) bool {
			return lp.LineI == prevPos2.LineI
		})
		for i := prevPos2I + 1; i < len(newT.Path.Follows) && newT.Path.Follows[i].LineI == newPos2.LineI; i++ {
			enx := EventEntryExit{
				LineI: newT.Path.Follows[i].LineI,
				Enter: false,
			}
			events = append(events, enx)
		}
		/*
			for portI := PortA; portI <= PortC; portI++ {
				port := s.g.Layout.GetPort(prevPos.LineI, portI)
				if port.Conn().LineI == newPos.LineI {
					if nextPort != -1 {
						zap.S().Fatalf("multiple ports connect to the same line")
					}
					nextPort = portI
				}
			}
		*/
		/*
			events := make([]Event, 0)
			for i := oldT.TrailerFront + 1; i <= newT.TrailerFront; i++ {
				enx := EventEntryExit{
					LineI: newT.Path.Follows[i].LineI,
					Enter: true,
				}
				events = append(events, enx)
			}
			for i := oldT.TrailerBack + 1; i <= newT.TrailerBack; i++ {
				enx := EventEntryExit{
					LineI: newT.Path.Follows[i].LineI,
					Enter: false,
				}
				events = append(events, enx)
			}
		*/
		// TODO: maybe order the EventEntryExits so its not grouped by TrailerFront and TrailerBack
		//       now:    [entry1, entry4,  exit2,  exit3]
		//       actual: [entry1,  exit2,  exit3, entry4]
		for _, event := range events {
			s.eventsS.Send(event)
		}
	}
	snap := s.g.snapshot()
	for i := range s.g.trains {
		t := snap.Trains[i]
		s.trainStates[i].Train = t
		s.trainStates[i].Position, _ = s.g.Model2.CurrentPosition3(&t, false)
	}
}

func (s *Simulator) checkTrainPower(i int) {
	t := s.g.trains[i]
	powers := make([]int, 0, t.TrailerFront-t.TrailerBack+1)
	maxPower := -1
	shortCircuit := false
	for i := t.TrailerBack; i <= t.TrailerFront; i++ {
		section := t.Path.Follows[i]
		ls := s.lineStates[section.LineI]
		if len(powers) > 0 && ls.Power > 0 && ls.Power != powers[len(powers)-1] {
			shortCircuit = true
		}
		powers = append(powers, ls.Power)
		if maxPower == -1 || maxPower < ls.Power {
			maxPower = ls.Power
		}
	}
	if shortCircuit {
		zap.S().Errorf("train %d: short-circuit (powers = %s)", i, powers)
	}
}

type ActorAndRef struct {
	Actor Actor
	Ref   ActorRef
}

// TODO: ValShortNotify
// TODO: ValCurrent
// TODO: applySwitch
// TODO: apply

type lineState struct {
	// Lock controls access to every field in this struct (except itself).
	Lock sync.Mutex

	Occupied bool
	Power    int

	// Below: for switches only
	SwitchState SwitchState
}

func (s *Simulator) logEvents() {
	ch := make(chan Event)
	s.events.Subscribe("logEvents", ch)
	defer s.events.Unsubscribe(ch)
	for ev := range ch {
		zap.S().Infof("new event: %s", ev)
	}
}
