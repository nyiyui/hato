package conn

import (
	"fmt"
	"strings"
	"time"

	"golang.org/x/exp/slices"
)

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
	State    STState
	Position int64 // um
	Velocity int64 // um/s
	// Monotonic is time from an arbitrary starting point. This should be used to compare with other ValAttitudes that come from the same device.
	Monotonic int64
	Time      time.Time
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
		"attitude(%d %vµm %vmm/s %vkm/h %v %c%c)",
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
	Start     time.Time
	Monotonic int64
	Sensors   []ValSeenSensor
	ID        []ValID
}

type ValSeenSensor struct {
	Name     string
	Seen     bool
	Position int64
}

func (v ValSeen) String() string {
	return fmt.Sprintf("seen(%d %#v)", v.Monotonic, v.Sensors)
}

type ValID struct {
	RFID []byte
}

type ValCurrent struct {
	Values []ValCurrentInner
}

func (v *ValCurrent) Sort() {
	slices.SortFunc(v.Values, func(a, b ValCurrentInner) bool {
		return a.Line < b.Line
	})
}

func (v ValCurrent) String() string {
	b := new(strings.Builder)
	fmt.Fprint(b, "current")
	for _, val := range v.Values {
		flow := '0'
		if val.Flow {
			flow = '1'
		}
		fmt.Fprintf(b, " %s%c", val.Line, flow)
	}
	return b.String()
}

type ValCurrentInner struct {
	Line LineName
	Flow bool
}

type ValShortNotify struct {
	Line      LineName
	Monotonic int64
}

func (s ValShortNotify) String() string {
	return fmt.Sprintf("SL%sT%d", s.Line, s.Monotonic)
}
