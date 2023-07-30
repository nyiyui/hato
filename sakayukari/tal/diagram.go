// Diagram - timetable for trains
package tal

import (
	"fmt"
	"log"

	"golang.org/x/exp/slices"
	. "nyiyui.ca/hato/sakayukari"
	"nyiyui.ca/hato/sakayukari/tal/layout"
)

type DiagramConf struct {
	Guide    ActorRef
	Model    ActorRef
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
	NextSegmentI int
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
	if tss.NextSegmentI == len(ts.Segments)-1 {
		return
	}
	if tss.NextSegmentI > len(ts.Segments)-1 {
		panic("tss.CurrentSegmentI overflow")
	}
	s := ts.Segments[tss.NextSegmentI]
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
	if dist >= 10*layout.Millimeter {
		return
	}
	log.Printf("t %#v", t)
	log.Printf("t.Path %#v", t.Path)
	log.Printf("s.Target %#v", s.Target)
	log.Printf("now.Position %#v", now.Position)
	log.Printf("dist %d", dist)
	log.Printf("=== REACHED CurrentSegmentI %d", tss.NextSegmentI)
	current := ts.Segments[tss.NextSegmentI]
	if current.After != nil {
		if d.state.TSs[current.After.TS].NextSegmentI < current.After.S {
			log.Printf("waiting on after")
			return
		}
	}
	d.nextSegment(tsi)
}

func (d *diagram) nextSegment(tsi int) {
	ts := d.conf.Schedule.TSs[tsi]
	tss := &d.state.TSs[tsi]
	if tss.NextSegmentI == len(ts.Segments)-1 {
		log.Printf("*** DONE")
		return
	}
	tss.NextSegmentI++
	d.apply(d.latestGS, tsi)
}

func (d *diagram) apply(prevGS GuideSnapshot, tsi int) {
	ts := d.conf.Schedule.TSs[tsi]
	s := d.conf.Schedule.TSs[tsi].Segments[d.state.TSs[tsi].NextSegmentI]
	y := prevGS.Layout
	t := prevGS.Trains[ts.TrainI]
	nt := Train{
		Power: d.conf.Schedule.TSs[tsi].Segments[d.state.TSs[tsi].NextSegmentI].Power,
		State: 0, // automatically copied from original by guide
	}
	{
		target := s.Target
		if target.Precise == 0 {
			target.Port = layout.PortA
		}
		log.Printf("### apply tsi %d target %#v (%#v)", tsi, s.Target, target)
		// TODO: Only accounting for CurrentBack→CurrentFront might miss trailers that e.g. RFID uses. Maybe somwhow include trailers in guide's info?
		lpsBack := y.MustFullPathTo(t.Path.Follows[t.CurrentBack], LinePort{target.LineI, target.Port})
		lpsFront := y.MustFullPathTo(t.Path.Follows[t.CurrentFront], LinePort{target.LineI, target.Port})
		//lpsBack := y.PathToInclusive(t.Path[t.CurrentBack].LineI, s.Target.LineI)
		//lpsFront := y.PathToInclusive(t.Path[t.CurrentFront].LineI, s.Target.LineI)
		log.Printf("### lpsBack %d -> %#v", t.Path.Follows[t.CurrentBack].LineI, lpsBack)
		log.Printf("### lpsFront %d -> %#v", t.Path.Follows[t.CurrentFront].LineI, lpsFront)
		// We have to include all currents in the new path.
		// The longer one will include both CurrentBack and CurrentFront regardless of direction.
		if len(lpsBack.Follows) == 1 || len(lpsFront.Follows) == 1 {
			log.Printf("### ALREADY THERE")
			return
		}
		if len(lpsBack.Follows) > len(lpsFront.Follows) {
			nt.Path = &lpsBack
			nt.CurrentBack = 0
			nt.CurrentFront = len(lpsBack.Follows) - len(lpsFront.Follows)
		} else if len(lpsFront.Follows) > len(lpsBack.Follows) {
			nt.Path = &lpsFront
			nt.CurrentBack = 0
			nt.CurrentFront = len(lpsFront.Follows) - len(lpsBack.Follows)
		} else {
			nt.Path = &lpsFront // shouldn't matter
			nt.CurrentBack = 0
			nt.CurrentFront = 0
			if t.CurrentBack != t.CurrentFront {
				panic(fmt.Sprintf("same-length path from two different LineIs: %d (back) and %d (front)", t.CurrentBack, t.CurrentFront))
			}
			if nt.CurrentBack < 0 || nt.CurrentFront < 0 || len(nt.Path.Follows) == 0 {
				panic("assert failed")
			}
		}
	}
	gtu := GuideTrainUpdate{
		TrainI: d.conf.Schedule.TSs[tsi].TrainI,
		Train:  nt,
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
