package tal

import (
	"fmt"
	"time"

	"golang.org/x/exp/slices"
)

type History struct {
	Spans []Span
}

func (h *History) AddSpan(s Span) {
	s.Time = time.Now()
	h.Spans = append(h.Spans, s)
}

type Span struct {
	Time  time.Time
	Power int
	// Velocity in µm/s.
	Velocity      int64
	VelocityKnown bool
	// Position moved (delta) in µm.
	Position      int64
	PositionKnown bool
}

type spanUsage struct {
	SpanI    int
	Power    int
	Velocity int64
	Duration time.Duration
}

func (h *History) Interspan(from, to time.Time) (Span, []spanUsage) {
	// Span.Position is the position at the end of this interspan
	startI := -1 + slices.IndexFunc(h.Spans, func(s Span) bool {
		return s.Time.After(from)
	})
	endI := slices.IndexFunc(h.Spans, func(s Span) bool {
		return s.Time.After(to)
	})
	cumDelta := to.Sub(from)
	cum := Span{Time: from, PositionKnown: true}
	sus := make([]spanUsage, 0)
	for i := startI; i < endI; i++ {
		span := h.Spans[i]
		delta := func(i int) time.Duration {
			if i == startI {
				return from.Sub(span.Time)
			}
			if i == endI {
				return to.Sub(span.Time)
			}
			prev := h.Spans[i-1]
			return span.Time.Sub(prev.Time)
		}(i)
		if !span.VelocityKnown && !span.PositionKnown {
			panic(fmt.Sprintf("Span %d must either have velocity or position known", i))
		}
		if !span.VelocityKnown {
			span.Velocity = span.Position * 1000 / delta.Milliseconds()
			span.VelocityKnown = true
		}
		if !span.PositionKnown {
			span.Position = span.Velocity * delta.Milliseconds() / 1000
			span.PositionKnown = true
		}

		cum.Position += span.Position
		sus = append(sus, spanUsage{
			SpanI:    i,
			Power:    span.Power,
			Velocity: span.Velocity,
			Duration: delta,
		})
	}
	cum.Velocity = cum.Position * 1000 / cumDelta.Milliseconds()
	cum.VelocityKnown = true
	return cum, sus
}

func (h *History) Correct(from, to time.Time, actual Span) []int64 {
	cum, sus := h.Interspan(from, to)
	speeds := make([]int64, MaxPower) // speed[100] = how much time was used with this speed during from-to in permille
	cumDelta := to.Sub(from)
	for _, su := range sus {
		speeds[su.Power] = su.Duration.Microseconds() / cumDelta.Milliseconds()
	}
	// TODO: polyfit (maybe 2 degrees) over speed to fill in gaps
	deltaVelocity := actual.Velocity - cum.Velocity
	delta := make([]int64, MaxPower)
	for speed, permille := range speeds {
		delta[speed] = deltaVelocity * permille / 1000
	}
	return delta
}
