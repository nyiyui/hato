package ctl2

import (
	"log"
	"time"

	. "nyiyui.ca/hato/sakayukari"
	"nyiyui.ca/hato/sakayukari/tal"
	"nyiyui.ca/hato/sakayukari/tal/layout"
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
		time.Sleep(3 * time.Second)
		for i := 0; true; i++ {
			log.Printf("loop %d", i)
			a.OutputCh <- Diffuse1{
				Origin: guide,
				Value: tal.GuideTrainUpdate{
					TrainI:       0,
					Target:       &layout.LinePort{gs.Layout.MustLookupIndex("C"), layout.PortB},
					Power:        60,
					PowerFilled:  true,
					SetRunOnLock: true,
					RunOnLock:    true,
				},
			}
			setPower(0, 60)
			waitUntilTrainIn(0, "C", 0)
			setPower(0, 0)
			time.Sleep(1000 * time.Millisecond)
			a.OutputCh <- Diffuse1{
				Origin: guide,
				Value: tal.GuideTrainUpdate{
					TrainI: 0,
					Target: &layout.LinePort{gs.Layout.MustLookupIndex("A"), layout.PortA},
				},
			}
			setPower(0, 60)
			waitUntilTrainIn(0, "A", 0)
			setPower(0, 0)
		}
	}()
	return a
}
