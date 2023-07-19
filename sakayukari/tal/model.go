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
			m.handleAttitude(diffuse)
		}
		if diffuse.Origin == m.conf.Guide {
			if gs, ok := diffuse.Value.(GuideSnapshot); ok {
				m.latestGS = gs
			} else if gc, ok := diffuse.Value.(GuideChange); ok {
				_ = gc
				/*
					t := gc.Snapshot.Trains[gc.TrainI]
					y := m.latestGS.Layout
					switch gc.Type {
					case ChangeTypeCurrentBack:
						lp := t.Path[t.CurrentBack]
						_ = lp
						panic("not implemented yet")
					case ChangeTypeCurrentFront:
						lp := t.Path[t.CurrentFront]
						var precise int64
						switch lp.PortI {
						case layout.PortA:
							// just entered through port B or C
							// TODO
							if t.CurrentFront == 0 {
								// TODO: harder to assert where the train is - was it just placed by a human? switched paths and CurrentBack/CurrentFront by Diagram? ignore this case, and rely on previous info if avail
								panic("ignore")
							} else {
								prevExitLP := t.Path[t.CurrentFront-1]
								precise = -int64(y.Lines[prevExitLP.LineI].GetPort(prevExitLP.PortI).Length)
							}
						case layout.PortB, layout.PortC:
							// just entered through port A
							precise = 0
						}
							// TODO: we have position of front rn, now get position of side A.
							switch t.Orient {
							case FormOrientA:
								// side A changed
							case FormOrientB:
								// side B changed
							}
							m.actor.OutputCh <- Diffuse1{Value: Attitude{
								TrainI: gc.TrainI,
								Time:   time.Now(),
								Position: layout.Position{
									LineI:   lp.LineI,
									Precise: precise,
								},
								PositionKnown: true,
							}}
					default:
						panic("invalid ChangeType")
					}
					// should get an Attitude struct as the Value (not enough to get GuideSnapshot as they don't specify which trains are exactly (i.e. on the edge of a line boundary) where)
					panic("not implemented yet")
				*/
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
