package plan

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	"nyiyui.ca/hato/sakayukari/tal"
	"nyiyui.ca/hato/sakayukari/tal/layout"
)

// QuadraticEdgePlan plans a trip with a quadratic acc/deceleration to a constant speed.
// QuadraticEdgePlan.Start.Position is ignored.
type QuadraticEdgePlan struct {
	Start        PointPlan
	Acceleration int64 // unit is µm/s²
	Velocity     int64
	End          PointPlan
	Deceleration int64 // unit is µm/s² (positive values only)
}

func distaceToVelocity(v1, v2, a int64) int64 {
	return (v2*v2 - v1*v1) * 1000 / (2 * a) / 1000
}

func (tp *TrainPlanner) QuadraticEdgePlan(qep QuadraticEdgePlan, etaCh chan<- time.Time) error {
	var port layout.PortI
	var fd tal.FormData
	var t tal.Train
	{
		gs := tp.p.g.SnapshotMux.Current()
		t = gs.Trains[tp.trainI]
		var ok bool
		fd, ok = tp.p.g.Model2.GetFormData(t.FormI)
		if !ok {
			return errors.New("FormData not found")
		}
		fd.UpdateRelation()
		if len(fd.Relation.Coeffs) == 0 {
			panic("panik")
		}
		_, ok = fd.Relation.SolveForX(float64(qep.Start.Velocity))
		if !ok {
			return fmt.Errorf("no power can be given to attain start velocity of %d µm/s (start velocity)", qep.Start.Velocity)
		}
		_, ok = fd.Relation.SolveForX(float64(qep.End.Velocity))
		if !ok {
			return fmt.Errorf("no power can be given to attain end velocity of %d µm/s (end velocity)", qep.End.Velocity)
		}
		_, ok = fd.Relation.SolveForX(float64(qep.Velocity))
		if !ok {
			return fmt.Errorf("no power can be given to attain end velocity of %d µm/s (intermediate velocity)", qep.Velocity)
		}
		switch qep.End.Position.Port {
		case layout.PortA:
			port = qep.End.Position.Port
		case layout.PortB, layout.PortC:
			// path has to contain qep.End.Position
			start := t.Path.Follows[t.TrailerBack] // back is arbitrary; it can be TrailerFront as well
			path := tp.p.g.Layout.MustFullPathTo(start, layout.LinePort{
				LineI: qep.End.Position.LineI,
				PortI: port,
			})
			beforeLast := path.Follows[len(path.Follows)-2]
			switch tp.p.g.Layout.GetPort(beforeLast).Conn().PortI {
			case layout.PortA:
				// B or C
				port = qep.End.Position.Port
			case layout.PortB, layout.PortC:
				port = layout.PortA
			}
		}
	}
	power, ok := fd.Relation.SolveForX(float64(qep.Start.Velocity))
	if !ok {
		panic(fmt.Sprintf("no power can be given to attain start velocity of %d µm/s (start velocity)", qep.Start.Velocity))
	}
	_, err := tp.p.g.TrainUpdate(tal.GuideTrainUpdate{
		TrainI: tp.trainI,
		Target: &layout.LinePort{
			LineI: qep.End.Position.LineI,
			PortI: port,
		},
		Power:        int(power),
		PowerFilled:  true,
		RunOnLock:    true,
		SetRunOnLock: true,
	})
	if err != nil {
		return fmt.Errorf("train update: %w", err)
	}
	stopCh := make(chan struct{}, 1)
	// TODO: eta
	var stopOnce sync.Once
	stop := func() {
		stopCh <- struct{}{}
	}
	defer stopOnce.Do(stop)

	targetOffset := tp.p.g.Layout.PositionToOffset(*t.Path, qep.End.Position)

	var lastAccelVel int64
	startAccelTime := time.Now()
	for range time.NewTicker(generalInterval).C {
		lastAccelVel = qep.Start.Velocity + qep.Acceleration*time.Since(startAccelTime).Milliseconds()/1000
		power, ok = fd.Relation.SolveForX(float64(lastAccelVel))
		if !ok {
			panic(fmt.Sprintf("no power can be given to attain end velocity of %d µm/s (intermediate velocity)", qep.Velocity))
		}
		_, err = tp.p.g.TrainUpdate(tal.GuideTrainUpdate{
			TrainI:      tp.trainI,
			Power:       int(power),
			PowerFilled: true,
		})
		if err != nil {
			defer stopOnce.Do(stop)
			zap.S().Errorf("TrainUpdate: %s", err)
		}

		gs := tp.p.g.SnapshotMux.Current()
		t := gs.Trains[tp.trainI]

		pos, _ := tp.p.g.Model2.CurrentPosition(&t)
		offset := tp.p.g.Layout.PositionToOffset(*t.Path, pos)
		stoppingDistance := distaceToVelocity(lastAccelVel, qep.End.Velocity, qep.Deceleration)
		zap.S().Infof("%d / %d | %s / %s (path: %s)", offset, targetOffset, pos, qep.End.Position, t.Path)
		if offset+stoppingDistance >= targetOffset {
			break
		}
	}
	var lastDecelVel int64
	startDecelTime := time.Now()
	for range time.NewTicker(generalInterval).C {
		lastDecelVel = lastAccelVel - qep.Deceleration*time.Since(startDecelTime).Milliseconds()/1000
		power, ok = fd.Relation.SolveForX(float64(lastDecelVel))
		if !ok {
			panic(fmt.Sprintf("no power can be given to attain end velocity of %d µm/s (intermediate velocity)", qep.Velocity))
		}
		_, err = tp.p.g.TrainUpdate(tal.GuideTrainUpdate{
			TrainI:      tp.trainI,
			Power:       int(power),
			PowerFilled: true,
		})
		if err != nil {
			defer stopOnce.Do(stop)
			zap.S().Errorf("TrainUpdate: %s", err)
		}
	}
	power, ok = fd.Relation.SolveForX(float64(qep.End.Velocity))
	if !ok {
		panic(fmt.Sprintf("no power can be given to attain end velocity of %d µm/s (end velocity)", qep.Velocity))
	}
	// set power just in case
	_, err = tp.p.g.TrainUpdate(tal.GuideTrainUpdate{
		TrainI:       tp.trainI,
		Power:        int(power),
		PowerFilled:  true,
		RunOnLock:    false, // reset RunOnLock
		SetRunOnLock: true,
	})
	if err != nil {
		return err
	}
	return nil
}
