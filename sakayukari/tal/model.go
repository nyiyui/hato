package tal

import (
	"fmt"
	"log"
	"time"

	. "nyiyui.ca/hato/sakayukari"
	"nyiyui.ca/hato/sakayukari/tal/cars"
	"nyiyui.ca/hato/sakayukari/tal/layout"
)

type Attitude struct {
	Time   time.Time
	TrainI int
	// Position of side A of the train.
	Position      layout.Position
	PositionKnown bool
	// Velocity of the train at that instant in µm/s.
	Velocity      int64
	VelocityKnown bool
}

func (a Attitude) String() string {
	return fmt.Sprintf("attitude(t%s p%#v v%d)", a.Time, a.Position, a.Velocity)
}

type ModelConf struct {
	Cars  cars.Data
	Guide ActorRef
	RFIDs []RFID
}

type RFID struct {
	Ref      ActorRef
	Position layout.Position
}

type model struct {
	conf     ModelConf
	actor    *Actor
	rfid     map[ActorRef]int
	latestGS GuideSnapshot
}

func Model(conf ModelConf) *Actor {
	a := &Actor{
		Comment:  "tal-model",
		InputCh:  make(chan Diffuse1),
		OutputCh: make(chan Diffuse1),
		Inputs:   make([]ActorRef, 0, 1+len(conf.RFIDs)),
		Type: ActorType{
			Input:       true,
			LinearInput: true,
			Output:      true,
		},
	}
	m := &model{
		conf:  conf,
		actor: a,
		rfid:  map[ActorRef]int{},
	}
	a.Inputs = append(a.Inputs, conf.Guide)
	for i, rfid := range conf.RFIDs {
		a.Inputs = append(a.Inputs, rfid.Ref)
		m.rfid[rfid.Ref] = i
	}
	go m.loop()
	return a
}

func (m *model) loop() {
	for diffuse := range m.actor.InputCh {
		if _, ok := diffuse.Value.(Attitude); ok {
			log.Printf("@@@ ATTITUDE %#v", diffuse.Value)
			m.handleAttitude(diffuse)
		}
		if diffuse.Origin == m.conf.Guide {
			if gs, ok := diffuse.Value.(GuideSnapshot); ok {
				m.latestGS = gs
			} else if gc, ok := diffuse.Value.(GuideChange); ok {
				log.Printf("@@@ MODEL diffuse %#v", diffuse)
				_ = gc
				t := gc.Snapshot.Trains[gc.TrainI]
				f, ok := m.conf.Cars.Forms[t.FormI]
				if !ok {
					panic(fmt.Sprintf("formation %s not found", t.FormI))
				}
				y := gc.Snapshot.Layout
				switch gc.Type {
				case ChangeTypeCurrentBack:
					lp := t.Path[t.CurrentBack]
					// Find the precise position in CurrentBack
					var precise int64 // how many µm away from port A?
					switch lp.PortI {
					case layout.PortA:
						precise = 0
					case layout.PortB, layout.PortC:
						precise = int64(y.Lines[lp.LineI].GetPort(lp.PortI).Length)
					}
					// Convert the position of back to position of side A.
					var pos layout.Position
					switch t.Orient {
					case FormOrientB:
						// Side A is the back, so we don't have to do anything.
						pos = layout.Position{LineI: lp.LineI, Precise: uint32(precise)}
					case FormOrientA:
						// Side B is the back, so move the position forwards by the length of the formation.
						var ok bool
						pos, ok = y.Traverse(t.Path[t.CurrentBack:], precise+int64(f.Length))
						if !ok {
							panic("Traverse failed")
						}
					}
					m.actor.OutputCh <- Diffuse1{Origin: Loopback, Value: Attitude{
						TrainI:        gc.TrainI,
						Time:          time.Now(),
						Position:      pos,
						PositionKnown: true,
					}}
				case ChangeTypeCurrentFront:
					lp := t.Path[t.CurrentFront]
					// Find the precise position in CurrentFront
					var precise int64 // how many µm away from port A?
					switch lp.PortI {
					case layout.PortA:
						// just entered through port B or C
						if t.CurrentFront == 0 {
							// NOTE: hard to assert where the train is - was it just placed by a human? switched paths and CurrentBack/CurrentFront by Diagram? ignore this case, and rely on previous info if avail
							log.Printf("ignore")
						} else {
							prevExitLP := t.Path[t.CurrentFront-1]
							exitPort := y.Lines[prevExitLP.LineI].GetPort(prevExitLP.PortI)
							if exitPort.ConnI != lp.LineI {
								panic("exitPort doesn't point to next LP in path")
							}
							if exitPort.ConnP != layout.PortB && exitPort.ConnP != layout.PortC {
								panic("exitPort says we enter CurrentFront from port A but we exit the CurrentFront line through port A according to path")
							}
							// the train front is basically at exitPort.ConnP
							precise = int64(y.Lines[exitPort.ConnI].GetPort(exitPort.ConnP).Length)
						}
					case layout.PortB, layout.PortC:
						// just entered through port A
						precise = 0
					}
					// Convert position of front to position of side A.
					var pos layout.Position
					switch t.Orient {
					case FormOrientA:
						// Side A is the front, so we don't have to do anything.
						pos = layout.Position{LineI: lp.LineI, Precise: uint32(precise)}
					case FormOrientB:
						// Side B is the front, so move the position backwards by the length of the formation.
						var ok bool
						pos, ok = y.Traverse(t.Path[:t.CurrentFront+1], -(precise + int64(f.Length)))
						if !ok {
							panic("Traverse failed")
						}
					}
					m.actor.OutputCh <- Diffuse1{Origin: Loopback, Value: Attitude{
						TrainI:        gc.TrainI,
						Time:          time.Now(),
						Position:      pos,
						PositionKnown: true,
					}}
				default:
					panic("invalid ChangeType")
				}
			}
		} else if _, ok := m.rfid[diffuse.Origin]; ok {
			//m.handleRFID(diffuse)
			panic("not implemented yet")
		} else {
			log.Printf("tal-model: unhandled diffuse %s", diffuse)
		}
	}
}

func (m *model) handleAttitude(diffuse Diffuse1) {
	log.Printf("handleAttitude %s", diffuse)
	panic("not implemented yet")
}

/*
func (m *model) handleRFID(diffuse Diffuse1) {
	ri := m.rfid[diffuse.Origin]
	seen := diffuse.Value.(conn.ValSeen)
	if len(seen.ID) != 1 {
		panic(fmt.Sprintf("got non-1-length ValSeen.ID: %#v", seen))
	}
	val := seen.ID[0]
	if len(val.RFID) != 7 {
		panic(fmt.Sprintf("got non-7-length RFID: %#v", val.RFID))
	}
	var fci cars.FormCarI
	var tagOffset uint32
	filled := false
OuterLoop:
	for fi, f := range m.conf.Cars.Forms {
		for ci, c := range f.Cars {
			if c.MifareID == *(*[7]byte)(val.RFID) {
				fci.Form = fi
				fci.Index = ci
				for ci2 := 0; ci2 < ci; ci2++ {
					c := f.Cars[ci2]
					tagOffset += c.Length
				}
				tagOffset += c.MifarePosition
				filled = true
				break OuterLoop
			}
		}
	}
	if !filled {
		panic(fmt.Sprintf("tal-model: unknown setcar: %#v", val.RFID))
	}
	f := m.conf.Cars.Forms[fci.Form]
	_ = f
	filled = false
	ti := -1
	for ti2, t := range m.latestGS.Trains {
		if t.FormI == fci.Form {
			ti = ti2
			filled = true
		}
	}
	if !filled {
		panic(fmt.Sprintf("tal-model: unknown train: formation %s", fci))
	}
	t := m.latestGS.Trains[ti]
	y := m.latestGS.Layout
	pos := m.conf.RFIDs[ri].Position
	rfidPathI := -1
	for i := t.CurrentBack; i <= t.CurrentFront; i++ {
		if t.Path[i].LineI == pos.LineI {
			// start from this LineI
			rfidPathI = i
		}
	}
	if rfidPathI == -1 {
		panic("LineI of RFID not in train's currents")
	}
	// TODO: consider a diff algo where we first convert the train to always make front == side A
	// ports A   BA   BB   A
	// lines |-0-||-1-||-2-|
	// rfid     x
	// train  ]====>
	// train  B    A
	// train  ]====>
	// train  A    B
	var tagToSideA int64
	switch t.Orient {
	case FormOrientA:
		tagToSideA = int64(tagOffset)
	case FormOrientB:
		tagToSideA = -int64(tagOffset)
	default:
		panic("invalid Train.FormOrient")
	}
	distRFIDPathIBackToTag := 0
	switch t.Path[rfidPathI].PortI {
	case layout.PortA:
		distRFIDPathIBackToTag = y.Lines[pos.LineI].Length - pos.Precise
	case layout.PortB, layout.PortC:
		distRFIDPathIBackToTag = pos.Precise
	default:
		panic("invalid PortI")
	}
	// ports A   BA   B
	// lines |-0-||-1-|
	// rfid     x
	// train  ]====>
	// train  B    A
	// ports B   AB   A
	// lines |-0-||-1-|
	// rfid     x
	// train  ]====>
	// train  B    A
	newPos := y.Traverse(t.Path[rfidPathI:], tagToSideA+distRFIDPathIBackToTag)
	a := Attitude{
		TrainI:        ti,
		Time:          time.Now(),
		Position:      newPos,
		PositionKnown: true,
	}
	m.actor.OutputCh <- Diffuse1{
		Origin: Loopback,
		Value:  a,
	}
}
*/
