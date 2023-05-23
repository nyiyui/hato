package yuuni

import (
	"encoding/json"
	"log"
	"time"

	"nyiyui.ca/soyuu/soyuuctl/conn"
	"nyiyui.ca/soyuu/soyuuctl/sakayukari"
)

const JyokouPower = 30

// JyokouPower is power required to move slowly

type calibrateState struct {
	Power     int
	Direction bool
	SARAfter  *time.Time
	// SARAfter says when to stop-and-reset (switch direction)
	Disabled bool
}

func calibrate(attitudeName, line, ticker string) sakayukari.Actor2 {
	t := time.Now()
	state := &calibrateState{
		Power:     50,
		Direction: true,
		SARAfter:  &t,
		Disabled:  true,
	}
	return sakayukari.Actor2{
		DependsOn: []sakayukari.ActorKey{attitudeName, ticker},
		UpdateFunc: func(self *sakayukari.Actor, gsm sakayukari.GraphStateMap, gs sakayukari.GraphState) (updated sakayukari.Value) {
			if state.Disabled {
				return sakayukari.SpecialIgnore
			}
			if state.SARAfter != nil && time.Now().After(*state.SARAfter) {
				state.Direction = !state.Direction
				state.SARAfter = nil
				state.Power++
				if state.Power == 255 {
					log.Print("CALIBRATE_END")
					state.Disabled = true
					return sakayukari.SpecialIgnore
				}
				log.Printf("CALIBRATE_NEXT %t %d", state.Direction, state.Power)
				//start next
				power := state.Power
				if !state.Direction {
					power = -power
				}
				return InputValue{Line: line, Value: power}
			}

			attitudeRaw := gs.States[gsm[attitudeName]]
			if attitudeRaw == nil {
				return sakayukari.SpecialIgnore
			}
			attitude := attitudeRaw.(conn.ValAttitude)
			if attitude.Front {
				// train is not "*almost past* breakbeam" yet
				// TODO: brittle
				return sakayukari.SpecialIgnore
			}
			b, err := json.Marshal(map[string]interface{}{
				"velocity": attitude.Velocity,
				"power":    state.Power,
				"time":     time.Now().Format(time.RFC3339),
			})
			if err != nil {
				panic(err)
			}
			log.Printf("CALIBRATE %s", b)
			{
				t := time.Now().Add(5 * time.Second)
				state.SARAfter = &t
			}
			return InputValue{
				Line:  line,
				Value: JyokouPower,
			}
		},
		SideEffects: true,
		Comment:     "calibrate",
	}
}
