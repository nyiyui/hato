package ctl

import (
	"fmt"
	"log"

	"github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	. "nyiyui.ca/hato/sakayukari"
	"nyiyui.ca/hato/sakayukari/conn"
	"nyiyui.ca/hato/sakayukari/runtime"
)

func Main() error {
	err := termui.Init()
	if err != nil {
		return fmt.Errorf("termui init: %w", err)
	}
	defer termui.Close()
	g := Graph{
		Actors: []Actor{
			uiEvents(),
			latestKey(ActorRef{Index: 0}),
		},
	}
	connState, connActors := conn.ConnActors([]conn.Id{
		conn.Id{Type: "soyuu-line-mega-0"},
		conn.Id{"soyuu-breakbeam", "itsybitsy0", "0"},
	})
	err = connState.Find()
	if err != nil {
		return fmt.Errorf("conn find: %w", err)
	}
	g.Actors = append(g.Actors, connActors...)
	g.Actors = append(g.Actors, conn.Velocity2(
		ActorRef{Index: 3},
		0,
	))
	// g.Actors = append(g.Actors, bodge.Model(bodge.ModelConf{
	// 	Attitudes: []bodge.AttitudeConf{
	// 		bodge.AttitudeConf{Source: ActorRef{Index: 4}},
	// 	},
	// }))
	g.Actors = append(g.Actors, Control(ActorRef{Index: 0}, ActorRef{Index: 2}, "A", "C", "D"))
	// g.Actors = append(g.Actors, bodge.Timing(ActorRef{Index: 2}, ActorRef{Index: 4}))
	//g.Actors = append(g.Actors, ui.ModelView(ActorRef{Index: 5}))

	i := runtime.NewInstance(&g)
	err = i.Check()
	if err != nil {
		return fmt.Errorf("check: %s", err)
	}
	err = i.Diffuse()
	if err != nil {
		return fmt.Errorf("diffuse: %s", err)
	}
	return nil
}

func uiEvents() Actor {
	a := Actor{
		Comment:  "uiEvents",
		OutputCh: make(chan Diffuse1),
		Type: ActorType{
			Output: true,
		},
	}
	go func() {
		for e := range termui.PollEvents() {
			if e.ID == "<C-c>" {
				log.Fatalf("interrupt") // TODO: cleanup when exiting
			}
			a.OutputCh <- Diffuse1{
				Value: UIEvent{e},
			}
		}
	}()
	return a
}

type UIEvent struct{ E termui.Event }

func (u UIEvent) String() string {
	return fmt.Sprint(u.E)
}

func latestKey(uiEvents ActorRef) Actor {
	a := Actor{
		Comment: "latestKey",
		InputCh: make(chan Diffuse1),
		Inputs:  []ActorRef{uiEvents},
		Type: ActorType{
			Input: true,
		},
	}
	state := widgets.NewParagraph()
	state.Text = "init"
	state.SetRect(0, 0, 10, 3)
	termui.Render(state)
	go func() {
		for e := range a.InputCh {
			state.Text = e.Value.(UIEvent).E.ID
			termui.Render(state)
		}
	}()
	return a
}
