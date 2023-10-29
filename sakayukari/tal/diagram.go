// Diagram - timetable for trains
package tal

import (
	"fmt"
	"log"

	"golang.org/x/exp/slices"
	. "nyiyui.ca/hato/sakayukari"
	. "nyiyui.ca/hato/sakayukari/prelude"
	"nyiyui.ca/hato/sakayukari/tal/layout"
)

type DiagramConf struct {
	Guide    ActorRef
	Model    ActorRef
	Sakuragi ActorRef
	Schedule Schedule
}

type diagram struct {
	conf     DiagramConf
	state    scheduleState
	actor    *Actor
	latestGS GuideSnapshot
}

type Schedule struct {
	TSs []TrainSchedule
}

type scheduleState struct {
	TSs []trainScheduleState
}

type TrainSchedule struct {
	TrainI   int
	Segments []Segment
}

type trainScheduleState struct {
	CurrentSegmentI int
	// minGeneration is the minimum acceptable Train.Generation. This is used to prevent working on outdated Train objects.
	minGeneration int
}

type Position = layout.Position

type SegmentI struct {
	TS int
	S  int
}

type Segment struct {
	// TODO: some way to establish a causal relationship with other Segments
	// Target is the target position for the train to go to by the time (above).
	// Note: the first Segment lists the starting position of the train (it is unspecified what Diagram will do if the train is not near (on the same Lines) that position).
	Target Position
	Power  int
	After  *SegmentI
}

func Diagram(conf DiagramConf) *Actor {
	a := &Actor{
		Comment:  "tal-diagram",
		InputCh:  make(chan Diffuse1),
		OutputCh: make(chan Diffuse1),
		Inputs:   []ActorRef{conf.Guide, conf.Model},
		Type: ActorType{
			Input:       true,
			LinearInput: true,
			Output:      true,
		},
	}
	d := &diagram{
		conf:  conf,
		actor: a,
		state: scheduleState{
			TSs: make([]trainScheduleState, len(conf.Schedule.TSs)),
		},
	}
	go d.loop()
	return a
}

func (d *diagram) loop() {
	firstApplied := false
	for diffuse := range d.actor.InputCh {
		switch {
		case diffuse.Origin == d.conf.Model:
			if _, ok := diffuse.Value.(Attitude); ok {
				d.handleAttitude(diffuse)
			}
		case diffuse.Origin == d.conf.Guide:
			if gs, ok := diffuse.Value.(GuideSnapshot); ok {
				d.latestGS = gs
				if !firstApplied {
					for tsi := range d.conf.Schedule.TSs {
						d.apply(d.latestGS, tsi)
						log.Printf("####################")
						log.Printf("firstApplied")
					}
					firstApplied = true
				}
			}
		}
	}
}

func (d *diagram) handleAttitude(diffuse Diffuse1) {
	now := diffuse.Value.(Attitude)
	if now == (Attitude{}) {
		return
	}
	if !now.PositionKnown {
		return
	}

	tsi := slices.IndexFunc(d.conf.Schedule.TSs, func(ts TrainSchedule) bool { return ts.TrainI == now.TrainI })
	if tsi == -1 {
		panic("handleAttitude unknown TrainSchedule")
	}
	ts := d.conf.Schedule.TSs[tsi]
	tss := &d.state.TSs[tsi]
	if tss.CurrentSegmentI == len(ts.Segments)-1 {
		return
	}
	if tss.CurrentSegmentI > len(ts.Segments)-1 {
		panic("tss.CurrentSegmentI overflow")
	}
	s := ts.Segments[tss.CurrentSegmentI]
	y := d.latestGS.Layout
	t := d.latestGS.Trains[ts.TrainI]
	if t.Generation < tss.minGeneration {
		return
	}
	follows := t.Path.Follows
	targetI := slices.IndexFunc(follows, func(lp LinePort) bool { return lp.LineI == s.Target.LineI })
	nowI := slices.IndexFunc(follows, func(lp LinePort) bool { return lp.LineI == now.Position.LineI })
	var dist int64
	if nowI == -1 {
		return
	}
	if targetI == -1 {
		log.Printf("att %#v", now)
		log.Printf("s %#v", s)
		log.Printf("targetI %d nowI %d", targetI, nowI)
		log.Printf("t %#v", t)
		log.Printf("t.Path %#v", t.Path)
		panic("targetI or nowI not found")
	}
	if targetI <= nowI {
		dist = -y.Count(follows[targetI:nowI+1], s.Target, now.Position)
	} else {
		dist = y.Count(follows[nowI:targetI+1], now.Position, s.Target)
	}
	d.actor.OutputCh <- Diffuse1{
		Origin: d.conf.Sakuragi,
		Value:  Message(fmt.Sprintf("next: dist %d mm; index %d", dist/1000, tss.CurrentSegmentI+1)),
	}
	if dist >= 10*layout.Millimeter {
		return
	}
	//log.Printf("t %#v", t)
	//log.Printf("t.Path %#v", t.Path)
	//log.Printf("s.Target %#v", s.Target)
	//log.Printf("now.Position %#v", now.Position)
	//log.Printf("dist %d", dist)
	//log.Printf("=== REACHED CurrentSegmentI %d", tss.CurrentSegmentI)
	//d.actor.OutputCh <- Diffuse2{
	//	Origin: d.conf.Sakuragi,
	//	Value:  Message(fmt.Sprintf("reached %d", tss.CurrentSegmentI)),
	//}
	current := ts.Segments[tss.CurrentSegmentI]
	if current.After != nil {
		if d.state.TSs[current.After.TS].CurrentSegmentI < current.After.S {
			log.Printf("waiting on after")
			return
		}
	}
	//d.nextSegment(tsi)
}

func (d *diagram) nextSegment(tsi int) {
	ts := d.conf.Schedule.TSs[tsi]
	tss := &d.state.TSs[tsi]
	if tss.CurrentSegmentI == len(ts.Segments)-1 {
		log.Printf("*** DONE")
		return
	}
	tss.CurrentSegmentI++
	d.apply(d.latestGS, tsi)
}

func (d *diagram) apply(prevGS GuideSnapshot, tsi int) {
	ts := d.conf.Schedule.TSs[tsi]
	s := d.conf.Schedule.TSs[tsi].Segments[d.state.TSs[tsi].CurrentSegmentI]
	t := prevGS.Trains[ts.TrainI]
	target := s.Target
	if target.Precise == 0 {
		target.Port = layout.PortA
	}
	gtu := GuideTrainUpdate{
		TrainI: d.conf.Schedule.TSs[tsi].TrainI,
		Target: &layout.LinePort{target.LineI, target.Port},
		// NOTE: for demo; for changing power via LiveControl
		//Power:       d.conf.Schedule.TSs[tsi].Segments[d.state.TSs[tsi].CurrentSegmentI].Power,
		//PowerFilled: true,
	}
	tss := &d.state.TSs[tsi]
	tss.minGeneration = t.Generation + 1
	//log.Printf("### apply (DRY-RUN) %#v", gtu)
	log.Printf("### apply %#v", gtu)
	d.actor.OutputCh <- Diffuse1{
		Origin: d.conf.Guide,
		Value:  gtu,
	}
	log.Printf("### APPLY DONE %#v", gtu)
}
