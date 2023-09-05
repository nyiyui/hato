package ctl

import (
	"time"

	"github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	. "nyiyui.ca/hato/sakayukari"
	"nyiyui.ca/hato/sakayukari/conn"
	"nyiyui.ca/hato/sakayukari/tal"
	"nyiyui.ca/hato/sakayukari/tal/layout"
)

func WaypointControl(uiEvents, guide ActorRef) Actor {
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
	go func() {
		waitUntil := func(f func() bool) {
			for {
				if f() {
					return
				}
				time.Sleep(100 * time.Millisecond)
			}
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
		speed := 80
		for len(gs.Trains) == 0 {
		}
		for {
			waitUntil(func() bool {
				t := gs.Trains[0]
				//return t.CurrentBack == t.CurrentFront && t.Path.Follows[t.CurrentBack].LineI == gs.Layout.MustLookupIndex("C")
				return t.Path.Follows[t.CurrentFront].LineI == gs.Layout.MustLookupIndex("C")
				//for i := t.CurrentBack; i <= t.CurrentFront; i++ {
				//	lp := t.Path.Follows[i]
				//	if lp.LineI == 2 {
				//		return true
				//	}
				//}
				//return false
			})
			for p := speed; p >= 13; p-- {
				setPower(0, p)
				time.Sleep(70 * time.Millisecond)
			}
			time.Sleep(1 * time.Second)
			a.OutputCh <- Diffuse1{
				Origin: guide,
				Value: tal.GuideTrainUpdate{
					TrainI: 0,
					Target: &layout.LinePort{gs.Layout.MustLookupIndex("nA"), layout.PortA},
				},
			}
			time.Sleep(10 * time.Second)
			for p := 13; p <= speed; p++ {
				setPower(0, p)
				time.Sleep(100 * time.Millisecond)
			}
			waitUntil(func() bool {
				t := gs.Trains[0]
				//return t.CurrentBack == t.CurrentFront && t.Path.Follows[t.CurrentBack].LineI == gs.Layout.MustLookupIndex("A")
				return t.Path.Follows[t.CurrentFront].LineI == gs.Layout.MustLookupIndex("A")
			})
			for p := speed; p >= 13; p-- {
				setPower(0, p)
				time.Sleep(70 * time.Millisecond)
			}
			time.Sleep(1 * time.Second)
			a.OutputCh <- Diffuse1{
				Origin: guide,
				Value: tal.GuideTrainUpdate{
					TrainI: 0,
					Target: &layout.LinePort{gs.Layout.MustLookupIndex("nC"), layout.PortA},
				},
			}
			time.Sleep(10 * time.Second)
			for p := 13; p <= speed; p++ {
				setPower(0, p)
				time.Sleep(100 * time.Millisecond)
			}
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
