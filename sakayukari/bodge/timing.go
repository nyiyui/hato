package bodge

import (
	"encoding/json"
	"log"
	"time"

	. "nyiyui.ca/hato/sakayukari"
	"nyiyui.ca/hato/sakayukari/conn"
)

type timingState struct {
	Power     int
	Direction bool
	Points    []timingPoint
}

type timingPoint struct {
	Power    int
	Attitude conn.ValAttitude
}

func Timing(line, velocity ActorRef) Actor {
	a := Actor{
		Comment:  "timing",
		InputCh:  make(chan Diffuse1),
		OutputCh: make(chan Diffuse1),
		Inputs:   []ActorRef{velocity},
		Type: ActorType{
			Input:  true,
			Output: true,
		},
	}
	state := new(timingState)
	state.Direction = true
	state.Power = 24
	state.Points = make([]timingPoint, 0, 70)
	go func() {
		time.Sleep(time.Second) // TODO: fix bodge (wait for conn find)
		log.Print("timing start")
		const l = "A"
		a.OutputCh <- Diffuse1{
			Origin: line,
			Value:  conn.ReqLine{l, false, state.Direction, 0},
		}
		time.Sleep(time.Second) // TODO: fix bodge (wait for conn find)
		for state.Power < 100 {
			log.Printf("power %d", state.Power)
			a.OutputCh <- Diffuse1{
				Origin: line,
				Value:  conn.ReqLine{l, false, state.Direction, uint8(state.Power)},
			}
		Again:
			select {
			case d := <-a.InputCh:
				switch d.Origin {
				case velocity:
					att := d.Value.(conn.ValAttitude)
					if att.Front {
						goto Again
					}
					state.Points = append(state.Points, timingPoint{
						Power:    state.Power,
						Attitude: att,
					})
					data, _ := json.Marshal(state.Points[len(state.Points)-1])
					log.Printf("bodge-timing-data %s", data)
					state.Power++
				default:
					goto Again
				}
			case <-time.After(700 * time.Second):
				state.Power++
			}
		}
		a.OutputCh <- Diffuse1{
			Origin: line,
			Value:  conn.ReqLine{l, true, false, 0},
		}
	}()
	return a
}
