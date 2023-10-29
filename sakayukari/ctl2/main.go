package ctl2

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	. "nyiyui.ca/hato/sakayukari"
	"nyiyui.ca/hato/sakayukari/conn"
	"nyiyui.ca/hato/sakayukari/kujo"
	"nyiyui.ca/hato/sakayukari/runtime"
	"nyiyui.ca/hato/sakayukari/sakuragi"
	"nyiyui.ca/hato/sakayukari/senri"
	"nyiyui.ca/hato/sakayukari/tal"
	"nyiyui.ca/hato/sakayukari/tal/cars"
	"nyiyui.ca/hato/sakayukari/tal/layout"
)

func Main() error {
	defer zap.S().Sync()
	level := zap.LevelFlag("log-level", zap.DebugLevel, "set log level")
	flag.Parse()
	cfg := zap.NewDevelopmentConfig()
	cfg.Level = zap.NewAtomicLevelAt(*level)
	dev, err := cfg.Build()
	if err != nil {
		panic(err)
	}
	zap.ReplaceGlobals(dev)

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
	log.Printf("finding devices…")
	err = connState.Find()
	if err != nil {
		return fmt.Errorf("conn find: %w", err)
	}
	g.Actors = append(g.Actors, connActors...)
	y, err := layout.InitTestbench6c()
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
		path3 := y.MustFullPathTo(
			layout.LinePort{y.MustLookupIndex("mitouc3"), layout.PortA},
			layout.LinePort{y.MustLookupIndex("mitouc3"), layout.PortB},
		)
		_, _, _ = path, path2, path3
		g2.InternalSetTrains([]tal.Train{
			tal.Train{
				Power:        0,
				CurrentBack:  0,
				CurrentFront: 0,
				State:        tal.TrainStateNextAvail,
				FormI:        uuid.MustParse("a7453d82-d52f-43ec-84d2-54dcea72f8c1"),
				Orient:       tal.FormOrientA,
				Path:         &path,
			},
			//tal.Train{
			//	Power:        0,
			//	CurrentBack:  0,
			//	CurrentFront: 0,
			//	State:        tal.TrainStateNextAvail,
			//	FormI:        uuid.MustParse("7b920d78-0c1b-49ef-ab2e-c1209f49bbc6"),
			//	Orient:       tal.FormOrientA,
			//	Path:         &path3,
			//},
		})
		g2.PublishSnapshot()
	}
	guide := ActorRef{Index: len(g.Actors) - 1}
	//g.Actors = append(g.Actors, WaypointControl(guide, g2))
	g.Actors = append(g.Actors, *sakuragi.Sakuragi(sakuragi.Conf{
		Guide:  guide,
		Guide2: g2,
	}))
	sakuragi := ActorRef{Index: len(g.Actors) - 1}
	_ = sakuragi

	log.Printf("starting kujo…")
	kujoServer := kujo.NewServer(g2)
	go func() {
		http.ListenAndServe("0.0.0.0:8001", kujoServer.Handler())
	}()

	go func() {
		log.Printf("starting runtime…")
		i := runtime.NewInstance(&g)
		err = i.Check()
		if err != nil {
			log.Fatalf("check: %s", err)
		}
		err = i.Diffuse()
		if err != nil {
			log.Fatalf("diffuse: %s", err)
		}
	}()

	go func() {
		time.Sleep(1 * time.Second)
		WaypointControl2(g2, kujoServer)
	}()

	log.Printf("starting senri…")
	err = senri.Main(g2)
	if err != nil {
		return fmt.Errorf("senri: %s", err)
	}
	return nil
}
