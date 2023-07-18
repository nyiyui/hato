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

type Position struct {
	LineI layout.LineI
	// Precise is the position from port A in µm.
	Precise uint32
}

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
	// TODO: applying here somehow hangs the program
	//for tsi := range d.conf.Schedule.TSs {
	//	d.apply(tsi)
	//}
	for diffuse := range d.actor.InputCh {
		switch {
		case diffuse.Origin == d.conf.Guide:
			gs := diffuse.Value.(GuideSnapshot)
			log.Printf("new snapshot %#v", gs)
			for tsi, ts := range d.conf.Schedule.TSs {
				tss := &d.state.TSs[tsi]
				apply := false
				log.Printf("tss %#v", tss)
				if tss.CurrentSegmentI == len(ts.Segments)-1 {
					continue
				} else {
					t := gs.Trains[ts.TrainI]
					si := tss.CurrentSegmentI + 1
					if si == len(ts.Segments) {
						log.Printf("siOverflow %d", si)
						continue
					}
					log.Printf("si %d", si)
					s := ts.Segments[si]
					for i := t.CurrentBack; i <= t.CurrentFront; i++ {
						log.Printf("for %d =? %d", s.Target.LineI, t.Path[i].LineI)
						if s.Target.LineI == t.Path[i].LineI {
							// TODO: precise position (maybe using traps (e.g. if train's position goes from 0 to 100, then trigger position of 50))
							tss.CurrentSegmentI = si
							log.Printf("### new CurrentSegmentI: %d", tss.CurrentSegmentI)
							apply = true
						}
					}
				}
				if apply {
					d.apply(gs, tsi)
				}
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
		log.Printf("### apply tsi %d target %#v", tsi, s.Target)
		lpsBack := y.PathTo(t.Path[t.CurrentBack].LineI, s.Target.LineI)
		lpsFront := y.PathTo(t.Path[t.CurrentFront].LineI, s.Target.LineI)
		log.Printf("### lpsBack %#v", lpsBack)
		log.Printf("### lpsFront %#v", lpsFront)
		// We have to include all currents in the new path.
		// The longer one will include both CurrentBack and CurrentFront regardless of direction.
		if len(lpsBack) > len(lpsFront) {
			nt.Path = lpsBack
			nt.CurrentBack = 0
			nt.CurrentFront = len(lpsBack) - len(lpsFront)
		} else if len(lpsFront) > len(lpsBack) {
			nt.Path = lpsFront
			nt.CurrentBack = len(lpsBack) - len(lpsFront)
			nt.CurrentFront = 0
		} else {
			nt.Path = lpsFront // shouldn't matter
			nt.CurrentBack = 0
			nt.CurrentFront = 0
			if t.CurrentBack != t.CurrentFront {
				panic(fmt.Sprintf("same-length path from two different LineIs: %d (back) and %d (front)", t.CurrentBack, t.CurrentFront))
			}
		}
	}
	gtu := GuideTrainUpdate{
		TrainI: d.conf.Schedule.TSs[tsi].TrainI,
		Train:  nt,
	}
	log.Printf("apply %#v", gtu)
	d.actor.OutputCh <- Diffuse1{
		Origin: d.conf.Guide,
		Value:  gtu,
	}
}
