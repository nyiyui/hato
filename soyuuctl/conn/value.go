package conn

import "fmt"

type Val interface {
	isVal()
	// Staleness() time.Time
	fmt.Stringer
}

type ValAttitude struct {
	State     STState
	Position  int64 // um
	Velocity  int64 // um/s
	Monotonic int64
	Certain   bool
}

func (_ ValAttitude) isVal() {}

func (v ValAttitude) String() string {
	return fmt.Sprintf("attitude(%d %v %vmm/s %v %t)", v.State, v.Position, float64(v.Velocity)/1000.0, v.Monotonic, v.Certain)
}

type ValSeen struct {
	Monotonic int64
	Sensor    string
	Seen      bool
}

func (_ ValSeen) isVal() {}

func (v ValSeen) String() string {
	return fmt.Sprintf("seen(%d %s %t)", v.Monotonic, v.Sensor, v.Seen)
}
