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
	CurrentSegmentI int
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
	att := diffuse.Value.(Attitude)
	if att == (Attitude{}) {
		return
	}
	//log.Printf("new Attitude %#v", att)
	tsi := slices.IndexFunc(d.conf.Schedule.TSs, func(ts TrainSchedule) bool { return ts.TrainI == att.TrainI })
	if tsi == -1 {
		panic("handleAttitude unknown TrainSchedule")
	}
	ts := d.conf.Schedule.TSs[tsi]
	tss := &d.state.TSs[tsi]
	apply := false
	//log.Printf("tss %#v", tss)
	if tss.CurrentSegmentI == len(ts.Segments)-1 {
		return
	} else {
		csi := tss.CurrentSegmentI
		if csi == len(ts.Segments) {
			log.Printf("siOverflow %d", csi)
			return
		}
		//log.Printf("si %d", si)
		s := ts.Segments[csi]
		//log.Printf("for %d =? %d", s.Target.LineI, t.Path[i].LineI)
		next := func() bool {
			if !att.PositionKnown {
				return false
			}
			y := d.latestGS.Layout
			t := d.latestGS.Trains[ts.TrainI]
			follows := t.Path.Follows
			a := slices.IndexFunc(follows, func(lp LinePort) bool { return lp.LineI == s.Target.LineI })
			b := slices.IndexFunc(follows, func(lp LinePort) bool { return lp.LineI == att.Position.LineI })
			var dist int64
			if a == -1 || b == -1 {
				log.Printf("att %#v", att)
				log.Printf("s %#v", s)
				log.Printf("a %d b %d", a, b)
				log.Printf("t %#v", t)
				log.Printf("t.Path %#v", t.Path)
				panic("a or b not found")
			}
			if a <= b {
				log.Printf("t.Path %#v", t.Path)
				log.Printf("Count(%#v, %#v, %#v)", follows[a:b+1], s.Target, att.Position)
				dist = -y.Count(follows[a:b+1], s.Target, att.Position)
			} else if a > b {
				_ = t
				//log.Printf("t.Path %#v", t.Path)
				//log.Printf("Count(%#v, %#v, %#v)", follows[b:a+1], att.Position, s.Target)
				dist = y.Count(follows[b:a+1], att.Position, s.Target)
			} else {
				panic("unreacheable")
			}
			if dist <= 300000 {
				log.Printf("att %#v", att)
				log.Printf("s %#v", s)
				log.Printf("a %d b %d", a, b)
				log.Printf("t %#v", t)
				log.Printf("t.Path %#v", t.Path)
				log.Printf("dist %d", dist)
				log.Printf("target %#v", s.Target)
				log.Printf("pos %#v", att.Position)
			}
			if dist > 10000 {
				return false
			}
			current := ts.Segments[tss.CurrentSegmentI]
			if current.After != nil {
				if d.state.TSs[current.After.TS].CurrentSegmentI < current.After.S {
					return false
				}
			}
			return true
		}()
		if next {
			// TODO: precise position (maybe using traps (e.g. if train's position goes from 0 to 100, then trigger position of 50))
			// TODO: implement After
			log.Printf("___ reached CurrentSegmentI: %d", tss.CurrentSegmentI)
			log.Printf("___ reached Segment: %#v", ts.Segments[tss.CurrentSegmentI])
			tss.CurrentSegmentI = csi + 1
			apply = true
		}
	}
	if apply {
		if tss.CurrentSegmentI == len(ts.Segments)-1 {
			log.Printf("*** DONE")
		} else {
			d.apply(d.latestGS, tsi)
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
	//log.Printf("### apply (DRY-RUN) %#v", gtu)
	log.Printf("### apply %#v", gtu)
	d.actor.OutputCh <- Diffuse1{
		Origin: d.conf.Guide,
		Value:  gtu,
	}
}
