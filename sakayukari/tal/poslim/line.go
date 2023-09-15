package poslim

import (
	"nyiyui.ca/hato/sakayukari/tal"
	"nyiyui.ca/hato/sakayukari/tal/layout"
)

type LineWitness struct {
	pl *PositionLimit
	g  *tal.Guide
}

func linePortToPos(y *layout.Layout, lp layout.LinePort) layout.Position {
	switch lp.PortI {
	case layout.PortA:
		return layout.Position{
			LineI: lp.LineI,
			Port:  lp.PortI,
		}
	case layout.PortB, layout.PortC:
		_, p := y.GetLinePort(lp)
		return layout.Position{
			LineI:   lp.LineI,
			Precise: p.Length,
			Port:    lp.PortI,
		}
	default:
		panic("unreachable")
	}
}

func (lw *LineWitness) recvChange(gc tal.GuideChange) Report {
	t := gc.Snapshot.Trains[gc.TrainI]
	var pos layout.Position
	switch gc.Type {
	case tal.ChangeTypeCurrentBack:
		pos = linePortToPos(gc.Snapshot.Layout, t.Path.AtIndex(t.CurrentBack-1))
	case tal.ChangeTypeCurrentFront:
		pos = linePortToPos(gc.Snapshot.Layout, t.Path.Follows[t.CurrentFront])
	default:
		panic("unreachable")
	}
	return Report{Trains: []ReportTrain{
		ReportTrain{
			TrainI:       gc.TrainI,
			Position:     pos,
			PositionType: PositionTypeAt,
		},
	}}
}

func (lw *LineWitness) recvSnapshot(gs tal.GuideSnapshot) Report {
	report := Report{Trains: make([]ReportTrain, 0, len(gs.Trains)*2)}
	for ti, t := range gs.Trains {
		report.Trains = append(report.Trains, ReportTrain{
			TrainI:       ti,
			Position:     linePortToPos(gs.Layout, t.Path.AtIndex(t.CurrentBack-1)),
			PositionType: PositionTypeAfter,
		})
		report.Trains = append(report.Trains, ReportTrain{
			TrainI:       ti,
			Position:     linePortToPos(gs.Layout, t.Path.Follows[t.CurrentFront]),
			PositionType: PositionTypeBefore,
		})
	}
	return report
}
