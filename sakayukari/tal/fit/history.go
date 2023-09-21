package fit

import (
	"time"

	"golang.org/x/exp/slices"
)

type History struct {
	TrainI int
	Spans  []Span
}

type Span struct {
	Time          time.Time
	Velocity      int64
	VelocityKnown bool
	Position      int64
	PositionKnown bool
}

func (h *History) Interspan(from, to time.Time) Span {
	// Span.Position is the position at the end of this interspan
	startI := -1 + slices.IndexFunc(h.Spans, func(s Span) {
		return s.Time.After(from)
	})
	endI := slices.IndexFunc(h.Spans, func(s Span) {
		return s.Time.After(to)
	})
	for i := startI; i < endI; i++ {
	}
}
