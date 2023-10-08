package ctl2

import (
	"log"
	"sync"
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
		}
	}()
	return a
}
