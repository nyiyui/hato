package conn

import "fmt"

// Integral length in micrometres.
type Length = int64

const (
	Lum int64 = 1
	Lmm int64 = 1000
	Lm  int64 = 1000000
)

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
	Front     bool
}

func (_ ValAttitude) isVal() {}

func (v ValAttitude) String() string {
	front := 'f'
	if !v.Front {
		front = 'b'
	}
	certain := 'y'
	if !v.Certain {
		certain = 'n'
	}
	return fmt.Sprintf(
		"attitude(%d %v %vmm/s %vkm/h %v %c%c)",
		v.State,
		v.Position,
		float64(v.Velocity)/1000.0,
		float64(v.Velocity*150)*3600/1e9,
		v.Monotonic,
		certain,
		front,
	)
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
