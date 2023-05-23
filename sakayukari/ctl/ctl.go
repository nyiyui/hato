package ctl

import (
	"time"

	"github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	. "nyiyui.ca/hato/sakayukari"
	"nyiyui.ca/hato/sakayukari/conn"
)

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
			}
			termui.Render(state)
		}
	}()
	return a
}
