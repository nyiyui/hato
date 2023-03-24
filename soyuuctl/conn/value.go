package conn

type Val interface {
	isVal()
	// Staleness() time.Time
}

type ValAttitude struct {
	State     STState
	Position  int64 // um
	Velocity  int64 // um/s
	Monotonic int64
	Certain   bool
}

func (_ ValAttitude) isVal() {}

type ValSeen struct {
	Monotonic int64
	Sensor    string
	Seen      bool
}

func (_ ValSeen) isVal() {}
