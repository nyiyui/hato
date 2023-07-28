package tal

import (
	"fmt"
	"log"
	"math/big"
	"time"

	"golang.org/x/exp/slices"

	. "nyiyui.ca/hato/sakayukari"
	"nyiyui.ca/hato/sakayukari/conn"
	"nyiyui.ca/hato/sakayukari/tal/cars"
	"nyiyui.ca/hato/sakayukari/tal/layout"
)

type Attitude struct {
	Time            time.Time
	TrainI          int
	TrainGeneration int
	// Position of side A of the train.
	Position      layout.Position
	PositionKnown bool
	// Velocity of the train at that instant in µm/s.
	// Positive velocity means the train is moving in the front direction (as defined by tal-guide).
	// TODO: change the meaning of positive velocity so positive = side A is the front
	Velocity      int64
	VelocityKnown bool
}

// TODO: List places where e.g. RFID Attitude is expected, and if tal-model doesn't receive an Attitude at that position and time (± error), reduce estimated velocity and position appropriately.

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
	conf            ModelConf
	actor           *Actor
	rfid            map[ActorRef]int
	latestGS        GuideSnapshot
	latestAttitudes []Attitude
	// currentAttitudes is the delta-updated attitudes derived from latestAttitudes.
	// This is to make sure the error is not going to increase every iteration of the handleDelta method.
	currentAttitudes []Attitude
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
	prev := time.Now()
	_ = prev
	for {
		select {
		case diffuse := <-m.actor.InputCh:
			if _, ok := diffuse.Value.(Attitude); ok {
				log.Printf("@@@ ATTITUDE %#v", diffuse.Value)
				m.handleAttitude(diffuse)
				//m.actor.OutputCh <- Diffuse1{Value: diffuse.Value}
				continue
			}
			if diffuse.Origin == m.conf.Guide {
				if gs, ok := diffuse.Value.(GuideSnapshot); ok {
					m.latestGS = gs
					// TODO: generate latestAttitudes
					if m.latestAttitudes != nil && len(m.latestAttitudes) != len(gs.Trains) {
						panic("adding/removing trains is not implemented yet")
						// general idea:
						//   adding trains - always appended
						//   removing trains - unsupported :)
					}
					if m.latestAttitudes == nil {
						m.latestAttitudes = make([]Attitude, len(gs.Trains))
					}
					if m.currentAttitudes == nil {
						m.currentAttitudes = make([]Attitude, len(gs.Trains))
					}
				} else if _, ok := diffuse.Value.(GuideChange); ok {
				}
			} else if _, ok := m.rfid[diffuse.Origin]; ok {
				m.handleRFID(diffuse)
			} else {
				log.Printf("tal-model: unhandled diffuse %s", diffuse)
			}
		default:
			now := time.Now()
			m.handleDelta(now, prev.Sub(now))
			prev = now
		}
	}
}

func (m *model) handleDelta(now time.Time, delta time.Duration) {
	for ti, la := range m.latestAttitudes {
		if la.VelocityKnown && la.PositionKnown {
			var yi int64
			{
				laDelta := now.Sub(la.Time)
				dt := new(big.Rat)
				dt.SetFrac64(laDelta.Nanoseconds(), 1e9)
				x := big.NewRat(la.Velocity, 1)
				x.Mul(x, dt)
				y := new(big.Float)
				y.SetRat(x)
				yi, _ = y.Int64()
			}
			// Handle x overflowing the Position.Precise.
			t := m.latestGS.Trains[ti]
			var path []LinePort
			{
				// t.Path has -1 as the last, but this is not useful - change Train.Path so that it contains a port for the last part as well
				i := slices.IndexFunc(t.Path.Follows, func(lp LinePort) bool { return lp.LineI == la.Position.LineI })
				if i == -1 {
					log.Printf("la: %#v", la)
					log.Printf("train: %#v", t)
					panic("la.Position.LineI nonexistent in Train.Path")
				}
				path = t.Path.Follows[i:]
				// Add la.Position.Precise
				lp := t.Path.Follows[i]
				switch lp.PortI {
				case layout.PortA:
					yi -= int64(la.Position.Precise)
				case layout.PortB, layout.PortC:
					yi += int64(la.Position.Precise)
				}
			}
			if yi < 0 {
				yi = -yi
				path = m.latestGS.Layout.ReverseFullPath(*t.Path).Follows
				i := slices.IndexFunc(path, func(lp LinePort) bool { return lp.LineI == la.Position.LineI })
				path = path[i:]
			}
			pos, ok := m.latestGS.Layout.Traverse(path, yi)
			if !ok {
				ca := m.currentAttitudes[ti]
				m.actor.OutputCh <- Diffuse1{Value: ca}
				//log.Printf("overflow")
				//log.Printf("yi %dµm", yi)
				//log.Printf("path %#v", path)
				//log.Printf("t %#v", t)
				continue
			}
			ca := m.currentAttitudes[ti]
			ca.TrainI = ti
			ca.TrainGeneration = t.Generation
			ca.Position = pos
			ca.PositionKnown = true
			ca.Velocity = la.Velocity
			ca.VelocityKnown = true
			m.currentAttitudes[ti] = ca
			m.actor.OutputCh <- Diffuse1{Value: ca}
		}
	}
}

func (m *model) attitudeOld(att Attitude) bool {
	return att.TrainGeneration < m.latestGS.Trains[att.TrainI].Generation
}

func (m *model) handleAttitude(diffuse Diffuse1) {
	log.Printf("handleAttitude %s", diffuse)
	att := diffuse.Value.(Attitude)
	prevAtt := m.latestAttitudes[att.TrainI]
	log.Printf("att %#v", att)
	log.Printf("prevAtt %#v", prevAtt)
	if !m.attitudeOld(prevAtt) && !m.attitudeOld(att) {
		log.Printf("not old")
		if !att.VelocityKnown && att.PositionKnown && prevAtt.PositionKnown {
			dt := att.Time.Sub(prevAtt.Time)
			// TODO: handle path being reset by Diagram
			path := m.latestGS.Trains[att.TrainI].Path.Follows
			startI := slices.IndexFunc(path, func(e LinePort) bool { return e.LineI == prevAtt.Position.LineI })
			endI := slices.IndexFunc(path, func(e LinePort) bool { return e.LineI == att.Position.LineI })
			log.Printf("path: %#v", path)
			log.Printf("startI: %#v", startI)
			log.Printf("endI: %#v", endI)
			log.Printf("prevAtt: %#v", prevAtt)
			log.Printf("att: %#v", att)
			if startI == -1 || endI == -1 {
				panic("Position not found, maybe Path was reset by Diagram?")
			}
			var dd int64
			var coeff int64
			if startI > endI {
				startI, endI = endI, startI
				//panic("startI > endI")
				log.Print("&&& startI > endI")
				coeff = -1
				dd = m.latestGS.Layout.Count(path[startI:endI+1], att.Position, prevAtt.Position)
			} else {
				coeff = +1
				dd = m.latestGS.Layout.Count(path[startI:endI+1], prevAtt.Position, att.Position)
			}
			x := new(big.Float)
			r := big.NewRat(dd, dt.Nanoseconds())
			r.Mul(r, big.NewRat(1e9, 1))
			x.SetRat(r)
			vel, _ := x.Int64()
			t := m.latestGS.Trains[att.TrainI]
			f, ok := m.conf.Cars.Forms[t.FormI]
			if !ok {
				panic(fmt.Sprintf("train %d %#v has unknown formation", att.TrainI, t))
			}
			if f.BaseVelocity != nil {
				m := f.BaseVelocity.M
				b := f.BaseVelocity.B
				att.Velocity = coeff * (m*int64(t.Power) + b)
				if att.Velocity < 0 {
					att.Velocity = 0
				}
				att.VelocityKnown = true
			} else {
				att.Velocity = coeff * vel
				att.VelocityKnown = true
			}
		}
		log.Printf("latestAttitude %s", att)
	}
	m.latestAttitudes[att.TrainI] = att
	m.actor.OutputCh <- Diffuse1{Value: att}
}

func (m *model) handleRFID(diffuse Diffuse1) {
	log.Printf("diffuse! %#v", diffuse)
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
	ti := slices.IndexFunc(m.latestGS.Trains, func(t Train) bool { return t.FormI == fci.Form })
	if ti == -1 {
		panic(fmt.Sprintf("tal-model: unknown train: formation %#v", fci))
	}
	t := m.latestGS.Trains[ti]
	y := m.latestGS.Layout
	pos := m.conf.RFIDs[ri].Position
	// TODO: track which cars are trailing and run IndexFunc in only the CurrentBack-CurrentFront + trailers part of t.Path
	rfidPathI := slices.IndexFunc(t.Path.Follows, func(lp LinePort) bool { return lp.LineI == pos.LineI })
	if rfidPathI == -1 {
		panic("LineI of RFID not in train's path")
	}
	// === Determine side A of the train
	// Consider the 4 scenarios:
	//               ↓ RFID sensor
	// Line:  A------R---B
	//        ↑ side A   ↑ side B
	// Train: B>r>A
	//        In the diagram above:
	//        - Orientation is A
	//        - The train goes towards the right
	//        - 'r' is where the RFID tag is
	// Note: displacement starts from point A of RFID, pointing in the direction of the front of the train
	// Scenario 1 (orient: A, front points towards: A):
	//        A<r<B
	//   A------R---B
	//   ^^^^^^     = - pos.Precise + tagOffset
	// Scenario 2 (orient: B, front points towards: A):
	//        B<r<A
	//   A------R---B
	//   ^^^^^^^^^^ = - pos.Precise - tagOffset
	// Scenario 3 (orient: A, front points towards: B/C):
	//        B>r>A
	//   A------R---B
	//   ^^^^^^^^^^ = + pos.Precise + tagOffset
	// Scenario 4 (orient: B, front points towards: B/C):
	//        A>r>B
	//   A------R---B
	//   ^^^^^^     = + pos.Precise - tagOffset
	var displacement int64
	{
		lp := t.Path.Follows[rfidPathI]
		precise := int64(pos.Precise)
		tagOffset := int64(tagOffset)
		switch lp.PortI {
		case layout.PortA:
			displacement -= precise
		case layout.PortB, layout.PortC:
			displacement += precise
		default:
			panic("invalid Train.Path[rfid-index].Port")
		}
		switch t.Orient {
		case FormOrientA:
			displacement += tagOffset
		case FormOrientB:
			displacement -= tagOffset
		default:
			panic("invalid Train.FormOrient")
		}
	}
	var path []LinePort
	if displacement < 0 {
		displacement = -displacement
		path = y.ReverseFullPath(*t.Path).Follows
		rfidPathI2 := slices.IndexFunc(path, func(lp LinePort) bool { return lp.LineI == pos.LineI })
		if rfidPathI2 == -1 {
			log.Printf("pos %#v", pos)
			log.Printf("path %#v", path)
			log.Printf("t.Path %#v", t.Path)
			log.Print("out of range: LineI of RFID not in train's currents")
			return
		}
		path = path[rfidPathI2:]
	} else {
		path = t.Path.Follows[rfidPathI:]
	}
	log.Printf("train %#v", t)
	log.Printf("tagOffset %d", tagOffset)
	log.Printf("pos %#v", pos)
	log.Printf("displacement %d", displacement)
	log.Printf("path %#v", path)
	sideAPos, ok := y.Traverse(path, displacement)
	if !ok {
		log.Print("out of range")
		return
	}
	a := Attitude{
		TrainI:          ti,
		TrainGeneration: t.Generation,
		Time:            time.Now(),
		Position:        sideAPos,
		PositionKnown:   true,
	}
	log.Printf("value! %#v", a)
	m.actor.OutputCh <- Diffuse1{
		Origin: Loopback,
		Value:  a,
	}
}
