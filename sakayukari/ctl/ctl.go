package ctl

import (
	"log"
	"sync"
	"time"

	"github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	. "nyiyui.ca/hato/sakayukari"
	"nyiyui.ca/hato/sakayukari/conn"
	"nyiyui.ca/hato/sakayukari/tal"
	"nyiyui.ca/hato/sakayukari/tal/layout"
)

func WaypointControl(uiEvents, guide ActorRef, g *tal.Guide) Actor {
	var gs tal.GuideSnapshot
	a := Actor{
		Comment:  "control",
		InputCh:  make(chan Diffuse1),
		OutputCh: make(chan Diffuse1),
		Inputs:   []ActorRef{uiEvents, guide},
		Type: ActorType{
			Input:       true,
			LinearInput: true,
			Output:      true,
		},
	}
	init := func(trainI int) {
		a.OutputCh <- Diffuse1{
			Origin: guide,
			Value: tal.GuideTrainUpdate{
				TrainI:       trainI,
				SetRunOnLock: true,
				RunOnLock:    true,
			},
		}
	}
	_ = init
	//init(0)
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
		waitUntilTrainOn := func(trainI int, comment string) {
			waitUntil(func() bool {
				t := gs.Trains[trainI]
				return t.Path.Follows[t.CurrentFront].LineI == gs.Layout.MustLookupIndex(comment)
			}, 0)
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
		smoothSpeed := func(ti, current, target int) {
			if current > target {
				for p := current; p >= target; p-- {
					setPower(ti, p)
					time.Sleep(160 * time.Millisecond)
				}
			} else {
				for p := current; p <= target; p++ {
					setPower(ti, p)
					time.Sleep(200 * time.Millisecond)
				}
			}
		}
		speed := 75
		_ = speed
		_ = smoothSpeed
		for len(gs.Trains) == 0 {
		}
		for {
			a.OutputCh <- Diffuse1{
				Origin: guide,
				Value: tal.GuideTrainUpdate{
					TrainI:       0,
					Target:       &layout.LinePort{gs.Layout.MustLookupIndex("B"), layout.PortB},
					SetRunOnLock: true,
					RunOnLock:    true,
				},
			}
			setPower(0, 60)
			a.OutputCh <- Diffuse1{
				Origin: guide,
				Value: tal.GuideTrainUpdate{
					TrainI:       1,
					Target:       &layout.LinePort{gs.Layout.MustLookupIndex("D"), layout.PortB},
					SetRunOnLock: true,
					RunOnLock:    true,
				},
			}
			_ = waitUntilTrainOn
			setPower(1, 60)
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				//waitUntilTrainOn(0, "B")
				//setPower(0, 30)
				//waitUntilTrainIn(0, "B", 4000*time.Millisecond)
				waitUntilTrainIn(0, "B", 0)
				for i := 0; i < 10; i++ {
					log.Printf("=== 0REACHED")
				}
				setPower(0, 0)
			}()
			wg.Add(1)
			go func() {
				defer wg.Done()
				//waitUntilTrainOn(1, "D")
				//setPower(1, 30)
				//waitUntilTrainIn(1, "D", 4000*time.Millisecond)
				waitUntilTrainIn(1, "D", 0)
				for i := 0; i < 10; i++ {
					log.Printf("=== 1REACHED")
				}
				setPower(1, 0)
			}()
			wg.Wait()
			time.Sleep(1000 * time.Millisecond)
			a.OutputCh <- Diffuse1{
				Origin: guide,
				Value: tal.GuideTrainUpdate{
					TrainI: 0,
					Target: &layout.LinePort{gs.Layout.MustLookupIndex("nC"), layout.PortB},
				},
			}
			setPower(0, 60)
			a.OutputCh <- Diffuse1{
				Origin: guide,
				Value: tal.GuideTrainUpdate{
					TrainI: 1,
					Target: &layout.LinePort{gs.Layout.MustLookupIndex("nA"), layout.PortB},
				},
			}
			setPower(1, 60)
			{
				var wg sync.WaitGroup
				wg.Add(1)
				go func() {
					defer wg.Done()
					waitUntilTrainIn(0, "C", 0)
					setPower(0, 0)
				}()
				wg.Add(1)
				go func() {
					defer wg.Done()
					waitUntilTrainIn(1, "A", 0)
					setPower(1, 0)
				}()
				wg.Wait()
			}
			panic("OK")
		}
	}()
	/*
		go func() {
			for range time.Tick(12 * time.Second) {
				a.OutputCh <- Diffuse1{
					Origin: guide,
					Value: tal.GuideTrainUpdate{
						TrainI: 0,
						Target: &layout.LinePort{0, layout.PortA},
					},
				}
				time.Sleep(6 * time.Second)
				a.OutputCh <- Diffuse1{
					Origin: guide,
					Value: tal.GuideTrainUpdate{
						TrainI: 0,
						Target: &layout.LinePort{2, layout.PortB},
					},
				}
			}
		}()
	*/
	go func() {
		power := 70
		for e := range a.InputCh {
			switch e.Origin {
			case guide:
				gs_, ok := e.Value.(tal.GuideSnapshot)
				if ok {
					gs = gs_
				}
			case uiEvents:
				key := e.Value.(UIEvent).E.ID
				switch key {
				case "Q":
					power += 2
					fallthrough
				case "q":
					power--
					a.OutputCh <- Diffuse1{
						Origin: guide,
						Value: tal.GuideTrainUpdate{
							TrainI:      0,
							Power:       power,
							PowerFilled: true,
						},
					}
				case "a":
					a.OutputCh <- Diffuse1{
						Origin: guide,
						Value: tal.GuideTrainUpdate{
							TrainI: 0,
							Target: &layout.LinePort{0, layout.PortA},
						},
					}
				case "c":
					a.OutputCh <- Diffuse1{
						Origin: guide,
						Value: tal.GuideTrainUpdate{
							TrainI: 0,
							Target: &layout.LinePort{2, layout.PortB},
						},
					}
				}
			}
		}
	}()
	return a
}

type controlState struct {
	Direction bool
}

func Control(uiEvents ActorRef, lineRef ActorRef, line, pointA, pointB string) Actor {
	a := Actor{
		Comment:  "control",
		InputCh:  make(chan Diffuse1),
		OutputCh: make(chan Diffuse1),
		Inputs:   []ActorRef{uiEvents},
		Type: ActorType{
			Input:  true,
			Output: true,
		},
	}
	cState := new(controlState)
	state := widgets.NewParagraph()
	state.Text = "init"
	state.SetRect(0, 0, 10, 3)
	termui.Render(state)
	go func() {
		for e := range a.InputCh {
			key := e.Value.(UIEvent).E.ID
			switch key {
			case "Q", "q":
				cState.Direction = key[0] == 'Q'
			case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9":
				a.OutputCh <- Diffuse1{Origin: lineRef, Value: conn.ReqLine{
					Line:      line,
					Direction: cState.Direction,
					Power:     (key[0] - '0') * 0x10,
				}}
				/*
					case "O", "o":
						a.OutputCh <- Diffuse1{Origin: lineRef, Value: conn.ReqSwitch{
							Line:      pointA,
							Direction: key[0] == 'O',
							Power:     0xff,
							Timeout:   1 * time.Second,
						}}
					case "P", "p":
						a.OutputCh <- Diffuse1{Origin: lineRef, Value: conn.ReqSwitch{
							Line:      pointB,
							Direction: key[0] == 'P',
							Power:     0xff,
							Timeout:   1 * time.Second,
						}}
				*/
			}
			termui.Render(state)
		}
	}()
	return a
}
