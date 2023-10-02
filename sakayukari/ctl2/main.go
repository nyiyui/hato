package ctl2

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/google/uuid"
	. "nyiyui.ca/hato/sakayukari"
	"nyiyui.ca/hato/sakayukari/conn"
	"nyiyui.ca/hato/sakayukari/kujo"
	"nyiyui.ca/hato/sakayukari/runtime"
	"nyiyui.ca/hato/sakayukari/tal"
	"nyiyui.ca/hato/sakayukari/tal/cars"
	"nyiyui.ca/hato/sakayukari/tal/layout"
)

func Main() error {
	g := Graph{
		Actors: []Actor{
			Actor{
				Comment:  "dummy",
				OutputCh: make(chan Diffuse1),
				Type:     ActorType{Output: true},
			},
		},
	}
	connState, connActors := conn.ConnActors([]conn.Id{
		conn.Id{"soyuu-kdss", "v4", "1"},
	})
	err := connState.Find()
	if err != nil {
		return fmt.Errorf("conn find: %w", err)
	}
	g.Actors = append(g.Actors, connActors...)
	y, err := layout.InitTestbench6()
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
	var g2 *tal.Guide
	{
		var actor Actor
		g2, actor = tal.NewGuide(tal.GuideConf{
			DontDemo: true,
			//Virtual: true,
			Layout: y,
			Actors: map[layout.LineID]ActorRef{
				layout.LineID{conn.Id{"soyuu-kdss", "v4", "1"}, "A"}: ActorRef{Index: 1},
				layout.LineID{conn.Id{"soyuu-kdss", "v4", "1"}, "B"}: ActorRef{Index: 1},
				layout.LineID{conn.Id{"soyuu-kdss", "v4", "1"}, "C"}: ActorRef{Index: 1},
				layout.LineID{conn.Id{"soyuu-kdss", "v4", "1"}, "D"}: ActorRef{Index: 1},
				layout.LineID{conn.Id{"soyuu-kdss", "v4", "1"}, "E"}: ActorRef{Index: 1},
				layout.LineID{conn.Id{"soyuu-kdss", "v4", "1"}, "F"}: ActorRef{Index: 1},
				layout.LineID{conn.Id{"soyuu-kdss", "v4", "1"}, "G"}: ActorRef{Index: 1},
				layout.LineID{conn.Id{"soyuu-kdss", "v4", "1"}, "H"}: ActorRef{Index: 1},
			},
			Cars: carsData,
		})
		g.Actors = append(g.Actors, actor)
		path := y.MustFullPathTo(
			layout.LinePort{y.MustLookupIndex("nagase1"), layout.PortA},
			layout.LinePort{y.MustLookupIndex("snb4"), layout.PortA},
		)
		path2 := y.MustFullPathTo(
			layout.LinePort{y.MustLookupIndex("snb4"), layout.PortA},
			layout.LinePort{y.MustLookupIndex("nagase1"), layout.PortA},
		)
		g2.InternalSetTrains([]tal.Train{
			tal.Train{
				Power:        0,
				CurrentBack:  0,
				CurrentFront: 0,
				State:        tal.TrainStateNextAvail,
				FormI:        uuid.MustParse("e5f6bb45-0abe-408c-b8e0-e2772f3bbdb0"),
				Orient:       tal.FormOrientA,
				Path:         &path,
			},
			tal.Train{
				Power:        0,
				CurrentBack:  0,
				CurrentFront: 0,
				State:        tal.TrainStateNextAvail,
				FormI:        uuid.MustParse("7b920d78-0c1b-49ef-ab2e-c1209f49bbc6"),
				Orient:       tal.FormOrientA,
				Path:         &path2,
			},
		})
	}
	guide := ActorRef{Index: len(g.Actors) - 1}
	g.Actors = append(g.Actors, WaypointControl(guide, g2))

	kujo := kujo.NewServer(g2)
	http.ListenAndServe("0.0.0.0:8001", kujo)

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
