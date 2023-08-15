package sim

import (
	"fmt"

	. "nyiyui.ca/hato/sakayukari"
	"nyiyui.ca/hato/sakayukari/conn"
	"nyiyui.ca/hato/sakayukari/runtime"
	"nyiyui.ca/hato/sakayukari/sakuragi"
	"nyiyui.ca/hato/sakayukari/tal"
	"nyiyui.ca/hato/sakayukari/tal/layout"
)

type Simulation struct {
	trains   []train
	latestGS tal.GuideSnapshot
	conf     SimulationConf
	actors   []Actor
}

type train struct {
	att tal.Attitude
}

type SimulationConf struct {
	Layout    *layout.Layout
	ModelConf tal.ModelConf
}

func New(conf SimulationConf) *Simulation {
	s := new(Simulation)
	s.conf = conf
	s.actors = []Actor{}
	return s
}

func (s *Simulation) setupRFID() {
	for ri := range s.conf.ModelConf.RFIDs {
		a := s.newRFID(ri)
		ai := len(s.actors)
		s.actors = append(s.actors, *a)
		s.conf.ModelConf.RFIDs[ri].Ref.Index = ai
	}
}

func (s *Simulation) setupLines() map[layout.LineID]ActorRef {
	s.actors = append(s.actors, *s.newLine(conn.Id{"soyuu-line", "v2", "yellow"}))
	yellow := ActorRef{Index: len(s.actors) - 1}
	s.actors = append(s.actors, *s.newLine(conn.Id{"soyuu-line", "v2", "white"}))
	white := ActorRef{Index: len(s.actors) - 1}
	res := map[layout.LineID]ActorRef{
		layout.LineID{conn.Id{"soyuu-line", "v2", "yellow"}, "A"}: yellow,
		layout.LineID{conn.Id{"soyuu-line", "v2", "yellow"}, "B"}: yellow,
		layout.LineID{conn.Id{"soyuu-line", "v2", "yellow"}, "C"}: yellow,
		layout.LineID{conn.Id{"soyuu-line", "v2", "yellow"}, "D"}: yellow,
		layout.LineID{conn.Id{"soyuu-line", "v2", "white"}, "A"}:  white,
		layout.LineID{conn.Id{"soyuu-line", "v2", "white"}, "B"}:  white,
		layout.LineID{conn.Id{"soyuu-line", "v2", "white"}, "C"}:  white,
		layout.LineID{conn.Id{"soyuu-line", "v2", "white"}, "D"}:  white,
	}
	return res
}

func (s *Simulation) setupGuideModel() {
	y := s.conf.Layout
	s.actors = append(s.actors, tal.Guide(tal.GuideConf{
		Layout: s.conf.Layout,
		Actors: s.setupLines(),
		Cars:   s.conf.ModelConf.Cars,
	}))
	guide := ActorRef{Index: len(s.actors) - 1}
	s.conf.ModelConf.Guide = guide
	s.actors = append(s.actors, *tal.Model(s.conf.ModelConf))
	model := ActorRef{Index: len(s.actors) - 1}
	s.actors = append(s.actors, *sakuragi.Sakuragi(sakuragi.Conf{
		Guide: guide,
		Model: model,
	}))
	s.actors = append(s.actors, *tal.Diagram(tal.DiagramConf{
		Guide: guide,
		Model: model,
		Schedule: tal.Schedule{
			TSs: []tal.TrainSchedule{
				{TrainI: 0, Segments: []tal.Segment{
					{tal.Position{y.MustLookupIndex("Z"), 0, layout.PortA}, 60, nil},
					{tal.Position{y.MustLookupIndex("W"), 0, layout.PortB}, 70, nil},
				}},
			},
		},
	}))
	a := Actor{
		Comment: "sim-latestGS",
		Inputs:  []ActorRef{guide},
		InputCh: make(chan Diffuse1),
		Type:    ActorType{Input: true, LinearInput: true},
	}
	go func() {
		for d := range a.InputCh {
			if d.Origin != guide {
				panic("latestGS from not-guide")
			}
			s.latestGS = d.Value.(tal.GuideSnapshot)
		}
	}()
	s.actors = append(s.actors, a)
}

func (s *Simulation) Run() {
	s.setupGuideModel()
	s.setupRFID()
	g := Graph{Actors: s.actors}
	i := runtime.NewInstance(&g)
	err := i.Check()
	if err != nil {
		panic(fmt.Sprintf("graph check: %s", err))
	}
	err = i.Diffuse()
	if err != nil {
		panic(fmt.Sprintf("graph diffuse: %s", err))
	}
	for {
		s.step()
	}
}
