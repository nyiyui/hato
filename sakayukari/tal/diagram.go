// Diagram - timetable for trains
package tal

import (
	"fmt"
	"log"

	. "nyiyui.ca/hato/sakayukari"
	"nyiyui.ca/hato/sakayukari/tal/layout"
)

type DiagramConf struct {
	Guide    ActorRef
	Schedule Schedule
}

type diagram struct {
	conf  DiagramConf
	state scheduleState
	actor *Actor
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
}

type Position = layout.Position

type Segment struct {
	// TODO: some way to establish a causal relationship with other Segments
	// Target is the target position for the train to go to by the time (above).
	// Note: the first Segment lists the starting position of the train (it is unspecified what Diagram will do if the train is not near (on the same Lines) that position).
	Target Position
	Power  int
}

func Diagram(conf DiagramConf) *Actor {
	a := &Actor{
		Comment:  "tal-diagram",
		InputCh:  make(chan Diffuse1),
		OutputCh: make(chan Diffuse1),
		Inputs:   []ActorRef{conf.Guide},
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
	for diffuse := range d.actor.InputCh {
		switch {
		case diffuse.Origin == d.conf.Guide:
			if _, ok := diffuse.Value.(GuideSnapshot); ok {
				d.handdleGS(diffuse)
			}
		}
	}
}

func (d *diagram) handdleGS(diffuse Diffuse1) {
	gs := diffuse.Value.(GuideSnapshot)
	//log.Printf("new snapshot %#v", gs)
	for tsi, ts := range d.conf.Schedule.TSs {
		tss := &d.state.TSs[tsi]
		apply := false
		//log.Printf("tss %#v", tss)
		if tss.CurrentSegmentI == len(ts.Segments)-1 {
			continue
		} else {
			t := gs.Trains[ts.TrainI]
			csi := tss.CurrentSegmentI
			if csi == len(ts.Segments) {
				log.Printf("siOverflow %d", csi)
				continue
			}
			//log.Printf("si %d", si)
			s := ts.Segments[csi]
			for i := t.CurrentBack; i <= t.CurrentFront; i++ {
				//log.Printf("for %d =? %d", s.Target.LineI, t.Path[i].LineI)
				if s.Target.LineI == t.Path[i].LineI {
					// TODO: precise position (maybe using traps (e.g. if train's position goes from 0 to 100, then trigger position of 50))
					log.Printf("___ reached CurrentSegmentI: %d", tss.CurrentSegmentI)
					log.Printf("___ reached Segment: %#v", ts.Segments[tss.CurrentSegmentI])
					tss.CurrentSegmentI = csi + 1
					apply = true
				}
			}
		}
		if apply {
			if tss.CurrentSegmentI == len(ts.Segments)-1 {
				log.Printf("*** DONE")
			} else {
				d.apply(gs, tsi)
			}
		}
	}
}

func (d *diagram) apply(prevGS GuideSnapshot, tsi int) {
	ts := d.conf.Schedule.TSs[tsi]
	s := d.conf.Schedule.TSs[tsi].Segments[d.state.TSs[tsi].CurrentSegmentI]
	y := prevGS.Layout
	t := prevGS.Trains[ts.TrainI]
	nt := Train{
		Power: d.conf.Schedule.TSs[tsi].Segments[d.state.TSs[tsi].CurrentSegmentI].Power,
		State: 0, // automatically copied from original by guide
	}
	{
		// TODO: bug happens when current is aleady on the goal; then the longer one is not necessarily the farthest?
		log.Printf("### apply tsi %d target %#v", tsi, s.Target)
		lpsBack := y.PathToInclusive(t.Path[t.CurrentBack].LineI, s.Target.LineI)
		lpsFront := y.PathToInclusive(t.Path[t.CurrentFront].LineI, s.Target.LineI)
		log.Printf("### lpsBack %d -> %#v", t.Path[t.CurrentBack].LineI, lpsBack)
		log.Printf("### lpsFront %d -> %#v", t.Path[t.CurrentFront].LineI, lpsFront)
		// We have to include all currents in the new path.
		// The longer one will include both CurrentBack and CurrentFront regardless of direction.
		if len(lpsBack) == 1 || len(lpsFront) == 1 {
			log.Printf("### ALREADY THERE")
			return
		}
		if len(lpsBack) > len(lpsFront) {
			nt.Path = lpsBack
			nt.CurrentBack = 0
			nt.CurrentFront = len(lpsBack) - len(lpsFront)
		} else if len(lpsFront) > len(lpsBack) {
			nt.Path = lpsFront
			nt.CurrentBack = 0
			nt.CurrentFront = len(lpsFront) - len(lpsBack)
		} else {
			nt.Path = lpsFront // shouldn't matter
			nt.CurrentBack = 0
			nt.CurrentFront = 0
			if t.CurrentBack != t.CurrentFront {
				panic(fmt.Sprintf("same-length path from two different LineIs: %d (back) and %d (front)", t.CurrentBack, t.CurrentFront))
			}
			if nt.CurrentBack < 0 || nt.CurrentFront < 0 || len(nt.Path) == 0 {
				panic("assert failed")
			}
		}
	}
	gtu := GuideTrainUpdate{
		TrainI: d.conf.Schedule.TSs[tsi].TrainI,
		Train:  nt,
	}
	//log.Printf("### apply (DRY-RUN) %#v", gtu)
	log.Printf("### apply %#v", gtu)
	d.actor.OutputCh <- Diffuse1{
		Origin: d.conf.Guide,
		Value:  gtu,
	}
}
