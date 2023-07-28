package conn

import (
	"bufio"
	"fmt"
	"log"
	"strings"
	"sync"

	. "nyiyui.ca/hato/sakayukari"
)

type handlerLine struct{}

type lineState struct {
	fileLock    sync.Mutex
	latestLines map[LineName]ReqLine
}

func (_ handlerLine) String() string {
	return "line"
}

func (_ handlerLine) HandleConn(a Actor, c *Conn) {
	reader := bufio.NewReader(c.F)
	_, err := fmt.Fprint(c.F, "gD087")
	if err != nil {
		log.Printf("%s: gD087: write line: %s", c.Path, err)
		return
	}
	state := new(lineState)
	state.latestLines = map[LineName]ReqLine{}
	go func() {
		for v := range a.InputCh {
			switch req := v.Value.(type) {
			case ReqLine:
				{
					latest, ok := state.latestLines[req.Line]
					if ok && latest == req {
						continue
					}
				}
				//log.Printf("ReqLine %s %s", c.Id, req)
				var err error
				func() {
					state.fileLock.Lock()
					defer state.fileLock.Unlock()
					_, err = fmt.Fprintf(c.F, "%s\n", req.String())
					b := make([]byte, 64000)
					b[0] = '_'
					b[len(b)-1] = '\n'
					_, err = c.F.Write(b)
				}()
				if err != nil {
					log.Printf("commit %s: %s", req, err)
				} else {
					state.latestLines[req.Line] = req
				}
			case ReqSwitch:
				log.Printf("ReqSwitch %s %s", c.Id, req)
				var err error
				func() {
					state.fileLock.Lock()
					defer state.fileLock.Unlock()
					_, err = fmt.Fprintf(c.F, "%s\n", req.String())
					b := make([]byte, 64000)
					b[0] = '_'
					b[len(b)-1] = '\n'
					_, err = c.F.Write(b)
				}()
				if err != nil {
					log.Printf("commit %s: %s", req, err)
				}
			default:
				log.Printf("unknown type %T", req)
			}
		}
	}()
	for {
		lineRaw, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("%s: read line: %s", c.Path, err)
			continue
		}
		if !strings.HasPrefix(lineRaw, " D") {
			continue
		}
		line := lineRaw[2:]
		switch line[0] {
		case 'C':
			line = line[1:]
			values, monotonic, err := parse(line)
			if err != nil {
				log.Printf("parse C: %s", err)
				continue
			}
			_ = monotonic
			v := ValCurrent{Values: make([]ValCurrentInner, 0, 4)}
			// NOTE: monotonic is in µs, not ms!
			for line, flow := range values {
				v.Values = append(v.Values, ValCurrentInner{
					Line: line,
					Flow: flow,
				})
			}
			// log.Printf("diffuse %s", v)
			a.OutputCh <- Diffuse1{Value: v}
			// log.Printf("diffuse DONE %s", v)
		case 'S':
			// example: " DSLAT16387"
			if line[1] != 'L' {
				log.Print("parse S: expected L")
				continue
			}
			target := line[2]
			_, monotonic, err := parse(line[3:])
			if err != nil {
				log.Printf("parse C: T: %s", err)
				continue
			}
			v := ValShortNotify{Line: LineName(target), Monotonic: monotonic}
			log.Printf("diffuseS %s", v)
			a.OutputCh <- Diffuse1{Value: v}
		}
	}
}

func (_ handlerLine) NewBlankActor() Actor {
	return Actor{
		Comment:  "blank handlerLine",
		InputCh:  make(chan Diffuse1),
		OutputCh: make(chan Diffuse1),
		Type: ActorType{
			Input:       true,
			LinearInput: true,
			Output:      true,
		},
	}
}
