package tal

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"time"

	"golang.org/x/exp/slices"
	"nyiyui.ca/hato/sakayukari/tal/layout"
)

type History struct {
	Spans []Span
	// TODO: support starting spans from different positions
	// SpanSets []SpanSet
}

func (h *History) AddSpan(s Span) {
	s.Time = time.Now()
	h.Spans = append(h.Spans, s)
}

func (h *History) TimeRange() (start, end time.Time, duration time.Duration) {
	last := len(h.Spans) - 1
	start = h.Spans[0].Time
	end = h.Spans[last].Time
	return start, end, end.Sub(start)
}

func (h *History) Clone() *History {
	spans := make([]Span, len(h.Spans))
	for i := range h.Spans {
		spans[i] = h.Spans[i]
	}
	return &History{Spans: spans}
}

type Span struct {
	Time time.Time // NOTE: non-monotonic-ness of ISO8601-formatted time shouldn't matter much here, as we're dealing with milliseconds, not nanoseconds

	Path    layout.FullPath
	SetPath bool

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

type Character struct {
	// Recorded is the start and end time when this data was taken.
	Recorded [2]time.Time
	// Points is the list of (speed, velocity) points
	Points [][2]int64
}

func (h *History) Character() Character {
	data, _ := json.Marshal(h)
	log.Printf("Character with %s", data)
	// History (h) must have known velocities only
	// use spanUsages (sus) to generate a list of (power, velocity) points
	// [(100, 123), (100, 456)] is ok (duplicate entries per power are ok)
	// return that data

	points := make([][2]int64, 0)
	prev := -1
	for i, span := range h.Spans {
		if !span.PositionKnown {
			continue
		}
		if prev != -1 && span.PositionKnown {
			// get weighted average of power
			var cum int64
			for j := prev; j < i; j++ {
				span2 := h.Spans[j]
				span3 := h.Spans[j+1]
				cum += int64(span2.Power) * span3.Time.Sub(span2.Time).Microseconds()
			}
			prevSpan := h.Spans[prev]
			total := span.Time.Sub(prevSpan.Time)
			if total.Milliseconds() == 0 {
				continue // don't set prev = i, so we treat this as a PositionKnown = false
			}
			power := cum / total.Microseconds()
			speed := (span.Position - prevSpan.Position) * 1000 / total.Milliseconds()
			if speed < 0 {
				panic(fmt.Sprintf("current span %d is behind previous span %d", i, prev))
			}
			points = append(points, [2]int64{power, speed})
			if speed < 50_000 { // debug
				log.Printf("point %d→%d: (%d, %d)", prev, i, power, speed)
				for j := prev; j < i; j++ {
					span2 := h.Spans[j]
					span3 := h.Spans[j+1]
					_, _ = span2, span3
					log.Printf("cum += %d", int64(span2.Power)*span3.Time.Sub(span2.Time).Microseconds())
				}
				log.Printf("total = %s", total)
				log.Printf("power = %d", power)
				log.Printf("speed = %d", speed)
			}
		}
		if span.PositionKnown {
			prev = i
		}
	}
	start, end, _ := h.TimeRange()
	return Character{
		Recorded: [2]time.Time{start, end},
		Points:   points,
	}
}

func ModelFromPoints(chars []Character) {
	// polyfit over that data (power, velocity)
	// return
}

func (h *History) Correct(from, to time.Time, actual Span) []int64 {
	cum, sus := h.Interspan(from, to)
	speeds := make([]int64, MaxPower) // speed[100] = how much time was used with speed 100 during from-to in permille
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

func evaluate(coeffs []float64, x float64) float64 {
	var y float64
	for i, coeff := range coeffs {
		y += math.Pow(x, float64(i)) * coeff
	}
	return y
}

// Extrapolate calculates the position at time at.
// TODO: explain algorithm
func (h *History) Extrapolate(relation Relation, at time.Time) int64 {
	// evaluate the spans
	var pos int64
	for i, span := range h.Spans {
		if i == 0 {
			continue
		}
		if span.Time.After(at) {
			break
		}
		prev := h.Spans[i-1]
		delta := span.Time.Sub(prev.Time)
		if prev.PositionKnown {
			pos = prev.Position
		}
		pos += int64(float64(delta.Milliseconds()) * evaluate(relation.Coeffs, float64(prev.Power)) / 1000)
		if span.PositionKnown {
			pos = span.Position
		}
	}
	// evaluate until at
	last := h.Spans[len(h.Spans)-1]
	delta := at.Sub(last.Time)
	pos += int64(float64(delta.Milliseconds()) * evaluate(relation.Coeffs, float64(last.Power)) / 1000)
	return pos
}
