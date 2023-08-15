package sim

import (
	"fmt"
	"time"

	"golang.org/x/exp/slices"
	. "nyiyui.ca/hato/sakayukari"
	"nyiyui.ca/hato/sakayukari/conn"
	"nyiyui.ca/hato/sakayukari/tal/cars"
	"nyiyui.ca/hato/sakayukari/tal/layout"
)

const rfidThreshold = 10000

func (s *Simulation) newRFID(i int) *Actor {
	a := &Actor{
		Comment:  fmt.Sprintf("sim-rfid-%d", i),
		OutputCh: make(chan Diffuse1),
		Type: ActorType{
			Output: true,
		},
	}
	self := s.conf.ModelConf.RFIDs[i]
	go func() {
		var prevData cars.MifareID
		now := time.Now()
		for ti, t := range s.trains {
			if !t.att.PositionKnown {
				continue
			}
			// for each RFID tag on a car, check if the tag overlaps the RFID sensor by a threshold
			// NOTE: if the value we want to send if the same as the previous value we sent, don't send (to match behaviour with soyuu-rfid)
			t2 := s.latestGS.Trains[ti]
			form := s.conf.ModelConf.Cars.Forms[t2.FormI]
			var carOffset int64
			for ci, car := range form.Cars {
				carOffset += int64(car.Length)
				if car.MifareID == (cars.MifareID{}) {
					continue
				}
				mifareOffset := carOffset - int64(car.Length) + int64(car.MifarePosition)
				pos, ok := s.conf.Layout.Traverse(t2.Path.Follows, mifareOffset)
				if !ok {
					panic(fmt.Sprintf("train %d car %d: mifare overflows path", ti, ci))
				}
				follows := t2.Path.Follows
				posI := slices.IndexFunc(follows, func(lp layout.LinePort) bool { return lp.LineI == pos.LineI })
				selfI := slices.IndexFunc(follows, func(lp layout.LinePort) bool { return lp.LineI == self.Position.LineI })
				if posI == -1 || selfI == -1 {
					panic(fmt.Sprintf("train %d car %d: pos or self not in path", ti, ci))
				}
				y := s.conf.Layout
				var delta int64
				if posI <= selfI {
					delta = y.Count(follows[posI:selfI+1], self.Position, pos)
				} else {
					delta = y.Count(follows[selfI:posI+1], pos, self.Position)
				}
				if delta > rfidThreshold {
					continue
				}
				data := car.MifareID
				if data == prevData {
					continue
				}
				a.OutputCh <- Diffuse1{Origin: Publish, Value: conn.ValSeen{
					Start: now,
					ID:    []conn.ValID{{RFID: data[:]}},
				}}
				prevData = data
			}
		}
	}()
	return a
}
