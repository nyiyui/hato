package fit

import "time"

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
