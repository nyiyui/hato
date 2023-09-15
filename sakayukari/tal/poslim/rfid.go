package poslim

import (
	"fmt"
	"log"

	"golang.org/x/exp/slices"
	"nyiyui.ca/hato/sakayukari/conn"
	"nyiyui.ca/hato/sakayukari/tal"
	"nyiyui.ca/hato/sakayukari/tal/cars"
	"nyiyui.ca/hato/sakayukari/tal/layout"
)

type RFIDWitness struct {
	pl  *PositionLimit
	pos layout.Position
}

func (rw *RFIDWitness) isWitness() {}

func (rw *RFIDWitness) Recv(seen conn.ValSeen) {
	if len(seen.ID) != 1 {
		panic(fmt.Sprintf("got non-1-length ValSeen.ID: %#v", seen))
	}
	val := seen.ID[0]
	if len(val.RFID) != 7 {
		panic(fmt.Sprintf("got non-7-length RFID: %#v", val.RFID))
	}

	forms := rw.pl.cars.Forms
	gs := rw.pl.g.LatestSnapshot()

	var fci cars.FormCarI
	{
		filled := false
	OuterLoop:
		for fi, f := range forms {
			for ci, c := range f.Cars {
				if c.MifareID == *(*[7]byte)(val.RFID) {
					fci.Form = fi
					fci.Index = ci
					filled = true
					break OuterLoop
				}
			}
		}
		if !filled {
			panic(fmt.Sprintf("tal-model: unknown form: %#v", val.RFID))
		}
	}
	f := forms[fci.Form]
	tagOffset := f.TagOffset(fci.Index)
	_ = f
	// TODO: make a ReportTrain for each train
	// steps:
	// - keep track of all PositionTypeAts and the train generation
	// - for each train:
	//   - if the train hasn't been PositionTypeAt-ed yet, choose the first RFID tag it would touch, and report that the train is before that
	//   - if the train has been PositionTypeAt-ed (not in this cycle)
	//     - if all RFID tags on the train have been at-ed return that it's after the latest PositionTypeAt
	//     - if RFID tags remain to be at-ed, say it is:
	//       - after the latest PositionTypeAt
	//       - before the next RFID tag
	tagTI := slices.IndexFunc(gs.Trains, func(t tal.Train) bool { return t.FormI == fci.Form })
	if tagTI == -1 {
		panic(fmt.Sprintf("tal-model: unknown train: formation %#v", fci))
	}
	report := Report{Trains: make([]ReportTrain, 0, len(gs.Trains))}
	for ti, t := range gs.Trains {
		pos := rw.pos
		y := gs.Layout

		rfidPathI := slices.IndexFunc(t.Path.Follows, func(lp layout.LinePort) bool { return lp.LineI == pos.LineI })
		if rfidPathI == -1 {
			report.Trains = append(report.Trains, ReportTrain{
				TrainI:       ti,
				PositionType: PositionTypeUnrelated,
			})
			continue
		}
		// TODO: track which cars are trailing and run IndexFunc in only the CurrentBack-CurrentFront + trailers part of t.Path
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
			case tal.FormOrientA:
				displacement += tagOffset
			case tal.FormOrientB:
				displacement -= tagOffset
			default:
				panic("invalid Train.FormOrient")
			}
		}
		var path []layout.LinePort
		if displacement < 0 {
			displacement = -displacement
			path = y.ReverseFullPath(*t.Path).Follows
			rfidPathI2 := slices.IndexFunc(path, func(lp layout.LinePort) bool { return lp.LineI == pos.LineI })
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
		_ = sideAPos
		report.Trains = append(report.Trains, ReportTrain{
			TrainI:       ti,
			PositionType: PositionTypeAt,
		})
		panic("TODO")
		//a := Attitude{
		//	TrainI:          ti,
		//	TrainGeneration: t.Generation,
		//	Time:            time.Now(),
		//	Position:        sideAPos,
		//	PositionKnown:   true,
		//}
		//log.Printf("value! %#v", a)
		//m.actor.OutputCh <- Diffuse1{
		//	Origin: Loopback,
		//	Value:  a,
		//}
	}
}
