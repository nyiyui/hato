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
		conn.Id{"soyuu-line", "v2", "yellow"},
		conn.Id{"soyuu-line", "v2", "white"},
		conn.Id{"soyuu-rfid", "adafruit:samd:adafruit_feather_m4", "0"},
		//conn.Id{"soyuu-line", "v2", "1"},
		//conn.Id{"soyuu-breakbeam", "itsybitsy0", "0"},
	})
	err = connState.Find()
	if err != nil {
		return fmt.Errorf("conn find: %w", err)
	}
	rfid0 := ActorRef{Index: len(g.Actors) + 2}
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
	y, err := layout.InitTestbench3()
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
	g.Actors = append(g.Actors, tal.Guide(tal.GuideConf{
		Layout: y,
		Actors: map[layout.LineID]ActorRef{
			layout.LineID{conn.Id{"soyuu-line", "v2", "yellow"}, "A"}: ActorRef{Index: 2},
			layout.LineID{conn.Id{"soyuu-line", "v2", "yellow"}, "B"}: ActorRef{Index: 2},
			layout.LineID{conn.Id{"soyuu-line", "v2", "yellow"}, "C"}: ActorRef{Index: 2},
			layout.LineID{conn.Id{"soyuu-line", "v2", "yellow"}, "D"}: ActorRef{Index: 2},
			layout.LineID{conn.Id{"soyuu-line", "v2", "white"}, "A"}:  ActorRef{Index: 3},
			layout.LineID{conn.Id{"soyuu-line", "v2", "white"}, "B"}:  ActorRef{Index: 3},
			layout.LineID{conn.Id{"soyuu-line", "v2", "white"}, "C"}:  ActorRef{Index: 3},
			layout.LineID{conn.Id{"soyuu-line", "v2", "white"}, "D"}:  ActorRef{Index: 3},
		},
		Cars: carsData,
	}))
	guide := ActorRef{Index: len(g.Actors) - 1}
	g.Actors = append(g.Actors, tal.GuideRender(guide))
	g.Actors = append(g.Actors, *tal.Model(tal.ModelConf{
		Guide: guide,
		Cars:  carsData,
		RFIDs: []tal.RFID{
			{rfid0, layout.Position{
				LineI:   y.MustLookupIndex("Y"),
				Precise: 252000,
				Port:    layout.PortB,
			}},
		},
	}))
	model := ActorRef{Index: len(g.Actors) - 1}
	g.Actors = append(g.Actors, *sakuragi.Sakuragi(sakuragi.Conf{
		Guide: guide,
		Model: model,
	}))
	//g.Actors = append(g.Actors, *tal.Diagram(tal.DiagramConf{
	//	Guide: guide,
	//	Model: model,
	//	Schedule: tal.Schedule{
	//		TSs: []tal.TrainSchedule{
	//			{TrainI: 0, Segments: []tal.Segment{
	//				{tal.Position{y.MustLookupIndex("Z"), 0, layout.PortA}, 60, nil},
	//				{tal.Position{y.MustLookupIndex("W"), 0, layout.PortB}, 70, nil},
	//				//{tal.Position{y.MustLookupIndex("Z"), 0, layout.PortA}, 60, nil},
	//				//{tal.Position{y.MustLookupIndex("W"), 0, layout.PortB}, 60, nil},
	//				//{tal.Position{y.MustLookupIndex("Z"), 0, layout.PortA}, 60, nil},
	//				//{tal.Position{y.MustLookupIndex("V"), 0}, 70, nil},
	//				//{tal.Position{y.MustLookupIndex("Z"), 0}, 125, nil},
	//				//{tal.Position{y.MustLookupIndex("W"), 0}, 100, nil},
	//				//{tal.Position{y.MustLookupIndex("Z"), 0}, 126, nil},
	//			}},
	//			//{TrainI: 1, Segments: []tal.Segment{
	//			//	{tal.Position{y.MustLookupIndex("Z"), 0}, 121,nil},
	//			//	{tal.Position{y.MustLookupIndex("V"), 0}, 100,nil},
	//			//	{tal.Position{y.MustLookupIndex("Z"), 0}, 123,nil},
	//			//	{tal.Position{y.MustLookupIndex("V"), 0}, 70,nil},
	//			//	{tal.Position{y.MustLookupIndex("Z"), 0}, 125,nil},
	//			//	{tal.Position{y.MustLookupIndex("V"), 0}, 100,nil},
	//			//	{tal.Position{y.MustLookupIndex("Z"), 0}, 126,nil},
	//			//}},
	//		},
	//	},
	//}))

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
