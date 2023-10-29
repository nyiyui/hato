package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	. "nyiyui.ca/hato/sakayukari"
	"nyiyui.ca/hato/sakayukari/ctl2"
	"nyiyui.ca/hato/sakayukari/kujo"
	"nyiyui.ca/hato/sakayukari/runtime"
	"nyiyui.ca/hato/sakayukari/sakuragi"
	"nyiyui.ca/hato/sakayukari/senri"
	"nyiyui.ca/hato/sakayukari/tal"
	"nyiyui.ca/hato/sakayukari/tal/cars"
	"nyiyui.ca/hato/sakayukari/tal/layout"
)

func main() {
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

	y, err := layout.InitTestbench6b()
	if err != nil {
		panic(err)
	}

	var carsData cars.Data
	{
		data, err := os.ReadFile("cars.json")
		if err != nil {
			zap.S().Fatalf("read cars.json: %w", err)
		}
		err = json.Unmarshal(data, &carsData)
		if err != nil {
			zap.S().Fatalf("parse cars.json: %w", err)
		}
	}

	g := Graph{
		Actors: []Actor{
			Actor{
				Comment:  "dummy",
				OutputCh: make(chan Diffuse1),
				Type:     ActorType{Output: true},
			},
		},
	}
	var g2 *tal.Guide
	var guideRef ActorRef
	s := tal.NewSimulator("command-line")
	{
		var actor Actor
		g2, actor = tal.NewGuide(tal.GuideConf{
			DontDemo: true,
			Layout:   y,
			Cars:     carsData,
		})
		g.Actors = append(g.Actors, actor)
		guideRef = ActorRef{Index: len(g.Actors) - 1}
		path := y.MustFullPathTo(
			layout.LinePort{y.MustLookupIndex("nagase1"), layout.PortA},
			layout.LinePort{y.MustLookupIndex("snb4"), layout.PortA},
		)
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
		})
		g2.PublishSnapshot()
	}
	s.SetGuide(g2)
	lineActors := s.GenActorRefs(func(a Actor) ActorRef {
		g.Actors = append(g.Actors, a)
		return ActorRef{Index: len(g.Actors) - 1}
	})
	g.Actors[guideRef.Index] = g2.RemakeActor(lineActors)

	g.Actors = append(g.Actors, *sakuragi.Sakuragi(sakuragi.Conf{
		Guide:  guideRef,
		Guide2: g2,
	}))
	sakuragiRef := ActorRef{Index: len(g.Actors) - 1}
	_ = sakuragiRef

	zap.S().Infof("starting kujo…")
	kujoServer := kujo.NewServer(g2)
	go http.ListenAndServe("0.0.0.0:8001", kujoServer.Handler())

	go func() {
		zap.S().Infof("starting runtime…")
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
		ctl2.WaypointControl2(g2, kujoServer)
	}()

	go func() {
		zap.S().Infof("starting simulation…")
		s.Run()
	}()

	zap.S().Infof("starting senri…")
	err = senri.Main(g2)
	if err != nil {
		zap.S().Fatalf("senri: %s", err)
	}
}
