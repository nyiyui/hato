package ctl

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	. "nyiyui.ca/hato/sakayukari"
	"nyiyui.ca/hato/sakayukari/conn"
	"nyiyui.ca/hato/sakayukari/runtime"
	"nyiyui.ca/hato/sakayukari/tal"
	"nyiyui.ca/hato/sakayukari/tal/layout"
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
		conn.Id{"soyuu-line", "v2", "4"},
		//conn.Id{"soyuu-line", "v2", "1"},
		//conn.Id{"soyuu-breakbeam", "itsybitsy0", "0"},
		//conn.Id{"soyuu-rfid", "adafruit:samd:adafruit_feather_m4", "0"},
	})
	err = connState.Find()
	if err != nil {
		return fmt.Errorf("conn find: %w", err)
	}
	g.Actors = append(g.Actors, connActors...)
	//g.Actors = append(g.Actors, conn.Velocity2(
	//	ActorRef{Index: 3},
	//	0,
	//))
	//g.Actors = append(g.Actors, bodge.Model(bodge.ModelConf{
	//	Attitudes: []bodge.AttitudeConf{
	//		bodge.AttitudeConf{Source: ActorRef{Index: 5}},
	//	},
	//}))
	// g.Actors = append(g.Actors, Control(ActorRef{Index: 0}, ActorRef{Index: 2}, "A", "C", "D"))
	// g.Actors = append(g.Actors, bodge.Timing(ActorRef{Index: 2}, ActorRef{Index: 4}))
	// g.Actors = append(g.Actors, ui.ModelView(ActorRef{Index: 6}))
	y, err := layout.InitTestbench2()
	if err != nil {
		panic(err)
	}
	data, _ := json.MarshalIndent(y, "", "  ")
	log.Printf("layout: %s", data)
	g.Actors = append(g.Actors, tal.Guide(tal.GuideConf{
		Layout: y,
		Actors: map[layout.LineID]ActorRef{
			layout.LineID{conn.Id{"soyuu-line", "v2", "4"}, "A"}: ActorRef{Index: 2},
			layout.LineID{conn.Id{"soyuu-line", "v2", "4"}, "B"}: ActorRef{Index: 2},
			layout.LineID{conn.Id{"soyuu-line", "v2", "4"}, "C"}: ActorRef{Index: 2},
			layout.LineID{conn.Id{"soyuu-line", "v2", "4"}, "D"}: ActorRef{Index: 2},
			//layout.LineID{conn.Id{"soyuu-line", "v2", "1"}, "A"}: ActorRef{Index: 3},
			//layout.LineID{conn.Id{"soyuu-line", "v2", "1"}, "B"}: ActorRef{Index: 3},
			//layout.LineID{conn.Id{"soyuu-line", "v2", "1"}, "C"}: ActorRef{Index: 3},
			//layout.LineID{conn.Id{"soyuu-line", "v2", "1"}, "D"}: ActorRef{Index: 3},
		},
	}))
	guide := ActorRef{Index: len(g.Actors) - 1}
	g.Actors = append(g.Actors, tal.GuideRender(guide))
	g.Actors = append(g.Actors, *tal.Diagram(tal.DiagramConf{
		Guide: guide,
		Schedule: tal.Schedule{
			TSs: []tal.TrainSchedule{
				{TrainI: 0, Segments: []tal.Segment{
					{tal.Position{y.MustLookupIndex("X"), 0}, 0},
					{tal.Position{y.MustLookupIndex("X"), 0}, 70},
					{tal.Position{y.MustLookupIndex("Y"), 0}, -70},
				}},
			},
		},
	}))

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
