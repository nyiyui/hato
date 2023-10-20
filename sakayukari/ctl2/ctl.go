package ctl2

import (
	"log"
	"sync"
	"time"

	"go.uber.org/zap"
	. "nyiyui.ca/hato/sakayukari"
	"nyiyui.ca/hato/sakayukari/audio"
	"nyiyui.ca/hato/sakayukari/tal"
	"nyiyui.ca/hato/sakayukari/tal/layout"
	"nyiyui.ca/hato/sakayukari/tal/layout/preset"
	"nyiyui.ca/hato/sakayukari/tal/layout/preset/kato"
	"nyiyui.ca/hato/sakayukari/tal/plan"
)

func WaypointControl(guide ActorRef, g *tal.Guide) Actor {
	var gs tal.GuideSnapshot
	a := Actor{
		Comment:  "control",
		InputCh:  make(chan Diffuse1),
		OutputCh: make(chan Diffuse1),
		Inputs:   []ActorRef{guide},
		Type: ActorType{
			Input:       true,
			LinearInput: true,
			Output:      true,
		},
	}
	go func() {
		for e := range a.InputCh {
			switch e.Origin {
			case guide:
				gs_, ok := e.Value.(tal.GuideSnapshot)
				if ok {
					gs = gs_
				}
			default:
				panic("unreachable")
			}
		}
	}()
	go func() {
		waitUntil := func(f func() bool, timeout time.Duration) {
			start := time.Now()
			for {
				if timeout != 0 && time.Since(start) > timeout {
					return
				}
				if f() {
					return
				}
				time.Sleep(100 * time.Millisecond)
			}
		}
		waitUntilTrainIn := func(trainI int, comment string, timeout time.Duration) {
			waitUntil(func() bool {
				t := gs.Trains[trainI]
				return t.CurrentFront == t.CurrentBack && t.Path.Follows[t.CurrentFront].LineI == gs.Layout.MustLookupIndex(comment)
				//return t.Path.Follows[t.CurrentFront].LineI == gs.Layout.MustLookupIndex(comment)
			}, timeout)
		}
		setPower := func(trainI, power int) {
			a.OutputCh <- Diffuse1{
				Origin: guide,
				Value: tal.GuideTrainUpdate{
					TrainI:      trainI,
					Power:       power,
					PowerFilled: true,
				},
			}
		}
		for len(gs.Trains) == 0 {
		}
		aPower := 90
		bPower := 90
		j, k := 0, 1
		for i := 0; true; i++ {
			log.Printf("loop %d", i)
			time.Sleep(500 * time.Millisecond)
			a.OutputCh <- Diffuse1{
				Origin: guide,
				Value: tal.GuideTrainUpdate{
					TrainI:       j,
					Target:       &layout.LinePort{gs.Layout.MustLookupIndex("mitouc2"), layout.PortB},
					Power:        12,
					PowerFilled:  true,
					SetRunOnLock: true,
					RunOnLock:    true,
				},
			}
			a.OutputCh <- Diffuse1{
				Origin: guide,
				Value: tal.GuideTrainUpdate{
					TrainI:       k,
					Target:       &layout.LinePort{gs.Layout.MustLookupIndex("mitouc3"), layout.PortA},
					Power:        12,
					PowerFilled:  true,
					SetRunOnLock: true,
					RunOnLock:    true,
				},
			}
			setPower(j, aPower)
			setPower(k, bPower)
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				waitUntilTrainIn(j, "mitouc2", 0)
				a.OutputCh <- Diffuse1{
					Origin: guide,
					Value: tal.GuideTrainUpdate{
						TrainI: j,
						Target: &layout.LinePort{gs.Layout.MustLookupIndex("snb4"), layout.PortA},
					},
				}
			}()
			wg.Add(1)
			go func() {
				defer wg.Done()
				waitUntilTrainIn(k, "mitouc3", 0)
				a.OutputCh <- Diffuse1{
					Origin: guide,
					Value: tal.GuideTrainUpdate{
						TrainI: k,
						Target: &layout.LinePort{gs.Layout.MustLookupIndex("nagase1"), layout.PortA},
					},
				}
			}()
			wg.Wait()
			go func() {
				ch := make(chan tal.GuideSnapshot, 0x10)
				g.SnapshotMux.Subscribe("ctl2", ch)
				defer g.SnapshotMux.Unsubscribe(ch)
				targetOffset := int64(2 * kato.S248)
				trainI := 0
				for range time.NewTicker(10 * time.Millisecond).C {
					gs := g.SnapshotMux.Current()
					t := gs.Trains[trainI]
					pos, overrun := g.Model2.CurrentPosition(&t)
					if overrun {
						break
					}
					offset := g.Layout.PositionToOffset(*t.Path, pos)
					if offset > targetOffset {
						break
					}
				}
				//audio.Play()
				zap.S().Infof("reached")
			}()
			{
				var wg sync.WaitGroup
				wg.Add(1)
				go func() {
					defer wg.Done()
					waitUntilTrainIn(j, "snb4", 0)
					time.Sleep(4 * time.Second)
					setPower(j, 12)
				}()
				wg.Add(1)
				go func() {
					defer wg.Done()
					waitUntilTrainIn(k, "nagase1", 0)
					time.Sleep(4 * time.Second)
					setPower(k, 12)
				}()
				wg.Wait()
			}
			time.Sleep(500 * time.Millisecond)
			//panic("TODO: save Train.History")
			j, k = k, j
			aPower, bPower = bPower, aPower
			if aPower > 255 {
				aPower = 50
			} else {
				aPower += 10
			}
			bPower = aPower
		}
	}()
	return a
}

func WaypointControl2(g *tal.Guide) {
	p := plan.NewPlanner(g)
	tp0 := p.NewTrainPlanner(0)
	tp1 := p.NewTrainPlanner(1)
	_ = tp1
	y := g.Layout
	etaCh := make(chan time.Time)
	go func() {
		timer := time.NewTimer(24 * time.Hour) // just some arbitraryily large #
		// TODO: fix (wrong code actually, need to call timer.Stop etc)
		go func() {
			var played bool
			for range timer.C {
				if !played {
					audio.Play()
					played = true
				}
			}
		}()
		for eta := range etaCh {
			zap.S().Infof("eta: %s %#v", eta.Sub(time.Now()), eta)
			d := eta.Sub(time.Now())
			d -= 3 * time.Second
			if d < 0 {
				d = 0
			}
			timer.Reset(d)
		}
	}()
	{
		pos := y.LinePortToPosition(layout.LinePort{LineI: y.MustLookupIndex("mitouc2"), PortI: layout.PortB})
		pos.Precise = kato.S248 + kato.S124 + 62_000
		err := tp0.LinearPlan(plan.LinearPlan{
			Start: plan.PointPlan{Velocity: preset.ScaleKmH(30)},
			End:   plan.PointPlan{Position: pos, Velocity: 0},
		}, etaCh)
		if err != nil {
			zap.S().Fatal(err)
		}
	}
	time.Sleep(3 * time.Second)
	{
		pos := y.LinePortToPosition(layout.LinePort{LineI: y.MustLookupIndex("nagase1"), PortI: layout.PortB})
		pos.Precise = kato.S248 + kato.S124
		err := tp0.LinearPlan(plan.LinearPlan{
			Start: plan.PointPlan{Velocity: preset.ScaleKmH(30)},
			End:   plan.PointPlan{Position: pos, Velocity: 0},
		}, etaCh)
		if err != nil {
			zap.S().Fatal(err)
		}
	}
}