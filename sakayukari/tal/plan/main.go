package plan

import (
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
	"nyiyui.ca/hato/sakayukari/tal"
	"nyiyui.ca/hato/sakayukari/tal/layout"
)

const generalInterval = 100 * time.Millisecond

type Planner struct {
	g *tal.Guide
}

func NewPlanner(g *tal.Guide) *Planner {
	return &Planner{g}
}

func (p *Planner) NewTrainPlanner(trainI int) *TrainPlanner {
	return &TrainPlanner{p, trainI}
}

type TrainPlanner struct {
	p      *Planner
	trainI int
}

type PointPlan struct {
	Position layout.Position
	Velocity int64
}

// LinearPlan plans a trip with a linear velocity to LinearPlan.End.
// LinearPlan.Start.Position is ignored.
type LinearPlan struct {
	Start PointPlan
	End   PointPlan
}

func (tp *TrainPlanner) LinearPlan(lp LinearPlan, etaCh chan<- time.Time) error {
	velocity := lp.Start.Velocity
	var powerStart, powerEnd float64
	var port layout.PortI
	{
		gs := tp.p.g.SnapshotMux.Current()
		t := gs.Trains[tp.trainI]
		fd, ok := tp.p.g.Model2.GetFormData(t.FormI)
		if !ok {
			return errors.New("FormData not found")
		}
		fd.UpdateRelation()
		if len(fd.Relation.Coeffs) == 0 {
			panic("panik")
		}
		powerStart, ok = fd.Relation.SolveForX(float64(velocity))
		if !ok {
			return fmt.Errorf("no power can be given to attain start velocity of %d µm/s (start velocity)", velocity)
		}
		powerEnd, ok = fd.Relation.SolveForX(float64(lp.End.Velocity))
		if !ok {
			return fmt.Errorf("no power can be given to attain end velocity of %d µm/s (end velocity)", lp.End.Velocity)
		}
		switch lp.End.Position.Port {
		case layout.PortA:
			port = lp.End.Position.Port
		case layout.PortB, layout.PortC:
			// path has to contain lp.End.Position
			start := t.Path.Follows[t.TrailerBack] // back is arbitrary; it can be TrailerFront as well
			path := tp.p.g.Layout.MustFullPathTo(start, layout.LinePort{
				LineI: lp.End.Position.LineI,
				PortI: port,
			})
			var beforeLast layout.LinePort
			if len(path.Follows) >= 2 {
				beforeLast = path.Follows[len(path.Follows)-2]
			} else {
				beforeLast = path.Start
			}
			switch tp.p.g.Layout.GetPort(beforeLast).Conn().PortI {
			case layout.PortA:
				// B or C
				port = lp.End.Position.Port
			case layout.PortB, layout.PortC:
				port = layout.PortA
			}
		}
	}
	newGeneration, err := tp.p.g.TrainUpdate(tal.GuideTrainUpdate{
		TrainI: tp.trainI,
		Target: &layout.LinePort{
			LineI: lp.End.Position.LineI,
			PortI: port,
		},
		Power:        int(powerStart),
		PowerFilled:  true,
		RunOnLock:    true,
		SetRunOnLock: true,
	})
	if err != nil {
		return fmt.Errorf("train update: %w", err)
	}
	stopCh := make(chan struct{}, 1)
	go func() {
		ch := make(chan tal.GuideSnapshot, 0x10)
		tp.p.g.SnapshotMux.Subscribe(fmt.Sprintf("LinearPlan for train %d", tp.trainI), ch)
		defer tp.p.g.SnapshotMux.Unsubscribe(ch)

		for {
			select {
			case gs := <-ch:
				t := gs.Trains[tp.trainI]
				if t.Generation < newGeneration {
					continue
				}
				targetOffset := tp.p.g.Layout.PositionToOffset(*t.Path, lp.End.Position)
				pos, _ := tp.p.g.Model2.CurrentPosition(&t)
				currentOffset := tp.p.g.Layout.PositionToOffset(*t.Path, pos)
				distance := targetOffset - currentOffset
				duration := distance * 1000 / velocity // use velocity from model
				eta := time.Now().Add(time.Duration(duration) * time.Millisecond)
				zap.S().Debugf("eta: %s", eta)
				etaCh <- eta
			case <-stopCh:
				return
			}
		}
	}()
	defer func() { stopCh <- struct{}{} }()

	{
		var targetOffset int64
		var targetOffsetSet bool
		for range time.NewTicker(generalInterval).C {
			gs := tp.p.g.SnapshotMux.Current()
			t := gs.Trains[tp.trainI]
			if t.Generation < newGeneration {
				continue
			}
			if !targetOffsetSet {
				targetOffset = tp.p.g.Layout.PositionToOffset(*t.Path, lp.End.Position)
				targetOffsetSet = true
			}

			pos, _ := tp.p.g.Model2.CurrentPosition(&t)
			offset := tp.p.g.Layout.PositionToOffset(*t.Path, pos)
			zap.S().Infof("%d / %d | %s / %s (path: %s)", offset, targetOffset, pos, lp.End.Position, t.Path)
			if offset >= targetOffset {
				break
			}
		}
	}
	_, err = tp.p.g.TrainUpdate(tal.GuideTrainUpdate{
		TrainI:       tp.trainI,
		Power:        int(powerEnd),
		PowerFilled:  true,
		RunOnLock:    false, // reset RunOnLock
		SetRunOnLock: true,
	})
	if err != nil {
		return err
	}
	return nil
}
