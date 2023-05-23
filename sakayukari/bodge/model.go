package bodge

import (
	"log"
	"time"

	. "nyiyui.ca/hato/sakayukari"
	"nyiyui.ca/hato/sakayukari/conn"
)

type ModelConf struct {
	Attitudes []AttitudeConf
}

type AttitudeConf struct {
	Source ActorRef
}

type modelState struct {
	Velocity     int64
	Position     int64
	PositionTime time.Time
}

func (m *modelState) GetCurrentPosition(now time.Time) int64 {
	return m.Position + now.Sub(m.PositionTime).Microseconds()*m.Velocity/1e6
}

func Model(conf ModelConf) Actor {
	a := Actor{
		Comment:  "model",
		InputCh:  make(chan Diffuse1),
		OutputCh: make(chan Diffuse1),
		Inputs:   make([]ActorRef, 0, len(conf.Attitudes)),
		Type: ActorType{
			Input:       true,
			LinearInput: true,
			Output:      true,
		},
	}
	for _, ac := range conf.Attitudes {
		a.Inputs = append(a.Inputs, ac.Source)
	}
	state := new(modelState)
	state.Velocity = -1
	go func() {
		for d := range a.InputCh {
			originI := -1
			for i, a := range conf.Attitudes {
				if a.Source == d.Origin {
					originI = i
				}
			}
			if originI == -1 {
				log.Printf("model: unknown Diffuse1 origin %s", d.Origin)
				continue
			}
			att := conf.Attitudes[originI]
			_ = att
			val := d.Value.(conn.ValAttitude)
			state.Velocity = val.Velocity
			state.Position = val.Position
			state.PositionTime = val.Time
			log.Printf("vel %v", state.Velocity)
			log.Printf("curPos %v", state.GetCurrentPosition(val.Time))
			a.OutputCh <- Diffuse1{Value: val}
		}
	}()
	go func() {
		for range time.NewTicker(1 * time.Millisecond).C {
			if state.Velocity == -1 {
				continue
			}
			now := time.Now()
			a.OutputCh <- Diffuse1{Value: conn.ValAttitude{
				Time:     now,
				Position: state.GetCurrentPosition(now),
				Velocity: state.Velocity,
			}}
		}
	}()
	return a
}
