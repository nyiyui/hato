package ctl

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	. "nyiyui.ca/hato/sakayukari"
	"nyiyui.ca/hato/sakayukari/conn"
	"nyiyui.ca/hato/sakayukari/runtime"
	"nyiyui.ca/hato/sakayukari/sakuragi"
	"nyiyui.ca/hato/sakayukari/tal"
	"nyiyui.ca/hato/sakayukari/tal/cars"
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
		conn.Id{"soyuu-line", "v2", "deepgreen"},
		conn.Id{"soyuu-line", "v2", "grey2"},
		//conn.Id{"soyuu-rfid", "adafruit:samd:adafruit_feather_m4", "0"},
		//conn.Id{"soyuu-rfid", "v2", "2"},
	})
	err = connState.Find()
	if err != nil {
		return fmt.Errorf("conn find: %w", err)
	}
	//rfid0 := ActorRef{Index: 4}
	//rfid1 := ActorRef{Index: 5}
	g.Actors = append(g.Actors, connActors...)
	y, err := layout.InitTestbench4()
	if err != nil {
		panic(err)
	}
	var carsData cars.Data
	{
		data, err := os.ReadFile("cars.json")
		if err != nil {
			return fmt.Errorf("read cars.json: %w", err)
		}
		err = json.Unmarshal(data, &carsData)
		if err != nil {
			return fmt.Errorf("parse cars.json: %w", err)
		}
	}
	data, _ := json.MarshalIndent(y, "", "  ")
	log.Printf("layout: %s", data)
	g.Actors = append(g.Actors, tal.NewGuide(tal.GuideConf{
		//Virtual: true,
		Layout: y,
		Actors: map[layout.LineID]ActorRef{
			layout.LineID{conn.Id{"soyuu-line", "v2", "deepgreen"}, "A"}: ActorRef{Index: 2},
			layout.LineID{conn.Id{"soyuu-line", "v2", "deepgreen"}, "B"}: ActorRef{Index: 2},
			layout.LineID{conn.Id{"soyuu-line", "v2", "deepgreen"}, "C"}: ActorRef{Index: 2},
			layout.LineID{conn.Id{"soyuu-line", "v2", "deepgreen"}, "D"}: ActorRef{Index: 2},
			layout.LineID{conn.Id{"soyuu-line", "v2", "grey2"}, "A"}:     ActorRef{Index: 3},
			layout.LineID{conn.Id{"soyuu-line", "v2", "grey2"}, "B"}:     ActorRef{Index: 3},
			layout.LineID{conn.Id{"soyuu-line", "v2", "grey2"}, "C"}:     ActorRef{Index: 3},
			layout.LineID{conn.Id{"soyuu-line", "v2", "grey2"}, "D"}:     ActorRef{Index: 3},
		},
		Cars: carsData,
	}))
	guide := ActorRef{Index: len(g.Actors) - 1}
	g.Actors = append(g.Actors, tal.GuideRender(guide))
	g.Actors = append(g.Actors, *tal.Model(tal.ModelConf{
		Guide: guide,
		Cars:  carsData,
		RFIDs: []tal.RFID{
			//{rfid0, layout.Position{
			//	LineI:   y.MustLookupIndex("B"),
			//	Precise: 0,
			//	Port:    layout.PortB,
			//}},
			//{rfid1, layout.Position{
			//	LineI:   y.MustLookupIndex("C"),
			//	Precise: 0,
			//	Port:    layout.PortB,
			//}},
		},
	}))
	model := ActorRef{Index: len(g.Actors) - 1}
	g.Actors = append(g.Actors, *sakuragi.Sakuragi(sakuragi.Conf{
		Guide: guide,
		Model: model,
	}))
	sakuragi := ActorRef{Index: len(g.Actors) - 1}
	_ = sakuragi
	//g.Actors = append(g.Actors, *tal.Diagram(tal.DiagramConf{
	//	Guide:    guide,
	//	Model:    model,
	//	Sakuragi: sakuragi,
	//	Schedule: tal.Schedule{
	//		TSs: []tal.TrainSchedule{
	//			{TrainI: 0, Segments: []tal.Segment{
	//				{tal.Position{y.MustLookupIndex("A"), 0, layout.PortA}, 60, nil},
	//				{tal.Position{y.MustLookupIndex("C"), 0, layout.PortB}, 70, nil},
	//			}},
	//		},
	//	},
	//}))
	g.Actors = append(g.Actors, WaypointControl(ActorRef{Index: 0}, guide))

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
