package tal

import "fmt"

// SwitchState a switch's state.
type SwitchState int

const (
	SwitchStateB      SwitchState = 1
	SwitchStateC      SwitchState = 2
	SwitchStateUnsafe SwitchState = 3
)

func (s SwitchState) String() string {
	switch s {
	case SwitchStateB:
		return "ssB"
	case SwitchStateC:
		return "ssC"
	case SwitchStateUnsafe:
		return "ssUnsafe"
	default:
		return fmt.Sprintf("%d", s)
	}
}

type switchClear struct {
	// LineI is the index of the line.
	LineI int
	// New state of the switch.
	State SwitchState
}

func (s switchClear) String() string {
	return fmt.Sprintf("switch-clear(%d-%d", s.LineI, s.State)
}
