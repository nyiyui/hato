package conn

import (
	"fmt"
	"log"
	"sync"
	"time"

	. "nyiyui.ca/hato/sakayukari"
)

type handlerLine struct{}

type lineState struct {
	fileLock sync.Mutex
}

func (_ handlerLine) HandleConn(a Actor, c *Conn) {
	state := new(lineState)
	for v := range a.InputCh {
		switch req := v.Value.(type) {
		case ReqLine:
			var err error
			func() {
				state.fileLock.Lock()
				defer state.fileLock.Unlock()
				_, err = fmt.Fprintf(c.F, "%s\n", req.String())
			}()
			if err != nil {
				log.Printf("commit %s: %s", req, err)
			}
		case ReqSwitch:
			afterCh := time.After(req.Timeout)
			go func() {
				<-afterCh
				var err error
				func() {
					state.fileLock.Lock()
					defer state.fileLock.Unlock()
					_, err = fmt.Fprintf(c.F, "%s\n", ReqLine{
						Line:  req.Line,
						Brake: req.BrakeAfter,
						Power: 0,
					}.String())
				}()
				if err != nil {
					log.Printf("commit(timeout) %s: %s", req, err)
				}
			}()
			req2 := ReqLine{
				Line:      req.Line,
				Direction: req.Direction,
				Power:     req.Power,
			}
			var err error
			func() {
				state.fileLock.Lock()
				defer state.fileLock.Unlock()
				_, err = fmt.Fprintf(c.F, "%s\n", req2.String())
			}()
			if err != nil {
				log.Printf("commit(switch) %s: %s", req, err)
			}
		}
	}
}

func (_ handlerLine) NewBlankActor() Actor {
	return Actor{
		Comment: "blank handlerLine",
		InputCh: make(chan Diffuse1),
		Type: ActorType{
			Input:       true,
			LinearInput: true,
		},
	}
}
