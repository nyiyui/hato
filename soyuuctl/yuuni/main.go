package yuuni

import (
	"fmt"
	"log"

	"github.com/gizak/termui/v3"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"nyiyui.ca/soyuu/soyuuctl/conn"
	"nyiyui.ca/soyuu/soyuuctl/sakayukari"
)

type EventValue struct {
	Value termui.Event
}

func (e EventValue) String() string { return fmt.Sprintf("%#v", e.Value) }

type lineState struct {
	power     uint8
	direction bool
}

type Yuuni struct {
	s *conn.State

	state *widgets.Paragraph

	lineState  *widgets.Paragraph
	lineStates map[string]*lineState
}

func (y *Yuuni) stHook(name string, v conn.Val) {
	switch v := v.(type) {
	case conn.ValAttitude:
		switch v.State {
		case conn.STStateSide:
			y.state.Text = fmt.Sprintf("side\n%d um\n%d um/s\nts %d", v.Position, v.Velocity, v.Monotonic)
		case conn.STStateTop:
			y.state.Text = "top"
		case conn.STStateBase:
			y.state.Text = "base"
		}
		ui.Render(y.state)
	}
}

func newYuuni() (*Yuuni, error) {
	s := conn.NewState()
	err := s.Find()
	if err != nil {
		log.Fatalf("find: %s", err)
	}
	state := widgets.NewParagraph()
	state.Text = "no state"
	state.SetRect(25, 0, 40, 5)
	ls := widgets.NewParagraph()
	ls.Text = "no line state"
	ls.SetRect(40, 0, 60, 5)
	return &Yuuni{
		s:         s,
		state:     state,
		lineState: ls,
		lineStates: map[string]*lineState{
			"A": new(lineState),
			"B": new(lineState),
		},
	}, nil
}

func Main() error {
	if err := ui.Init(); err != nil {
		return fmt.Errorf("init termui: %w", err)
	}
	defer ui.Close()

	y, err := newYuuni()
	if err != nil {
		return err
	}

	log.Print("waiting")
	y.s.SetupDone.Wait()

	kbEventsChan := make(chan sakayukari.Value)
	keyboardEvents := sakayukari.Actor2{
		RecvChan:    kbEventsChan,
		SideEffects: true,
	}

	// hysteresisA := hysteresis(1000)
	// hysteresisA.DependsOn = []string{"soyuu-breakbeam/itsybitsy0-0:A"}
	// hysteresisB := hysteresis(1000)
	// hysteresisB.DependsOn = []string{"soyuu-breakbeam/itsybitsy0-0:B"}

	g2 := sakayukari.Graph2{
		Actors: map[string]sakayukari.Actor2{
			"ui-breakbeam": breakbeam(map[string]string{
				"rA": "soyuu-breakbeam/itsybitsy0-0:A",
				"rB": "soyuu-breakbeam/itsybitsy0-0:B",
				// "hA": "breakbeam-A",
				// "hB": "breakbeam-B",
				"at": "attitude",
			}),
			// "breakbeam-A": hysteresisA,
			// "breakbeam-B": hysteresisB,
			"attitude": velocity(
				"soyuu-breakbeam/itsybitsy0-0:A",
				"soyuu-breakbeam/itsybitsy0-0:B",
			),
			"keyboard": keyboardEvents,
		},
	}
	for key, actor := range y.s.Actors() {
		actor := actor
		log.Printf("key %s actor %#v", key, actor)
		actor2 := sakayukari.Actor2{
			RecvChan:    actor.RecvChan,
			SideEffects: actor.SideEffects,
		}
		if actor.UpdateFunc != nil {
			actor2.UpdateFunc = func(self *sakayukari.Actor, _ sakayukari.GraphStateMap, gs sakayukari.GraphState) sakayukari.Value {
				return actor.UpdateFunc(self, gs)
			}
		}
		g2.Actors[key] = actor2
	}
	//log.Printf("g2 %#v", g2.Actors)
	/*
		2023/04/03 09:06:51 connecting to /dev/ttyACM0
		2023/04/03 09:06:51 g2 map[string]sakayukari.Actor2{"ui-breakbeam":sakayukari.Actor2{DependsOn:[]string{"soyuu-breakbeam/itsybitsy0-0"}, UpdateFunc:(func(map[string]int, sakayukari.GraphState) sakayukari.Value)(0xe64f0), RecvChan:(chan sakayukari.Value)(nil), SideEffects:true}}
		2023/04/03 09:06:51 executing graph
		2023/04/03 09:06:51 listening for [-1]
	*/
	g := g2.Convert()
	err = g.Check()
	if err != nil {
		log.Fatalf("check: %s", err)
	}
	log.Printf("executing graph")
	go g.Exec()
	for e := range ui.PollEvents() {
		switch e.ID {
		case "<C-c>":
			return nil
		default:
			kbEventsChan <- EventValue{e}
		}
	}
	return nil
}
