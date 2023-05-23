package model

type LineID string

// Position is a set position specific to each train and line.
type Position struct {
	Line   LineID
	Offset int64
	// Offset from zero position of line in µm
}

type Line struct {
	Type      LineType
	OneLength int64
	// OneLength is the length of a one_ or sw12 (main line) line in µm
	OneRadius int64
	// OneRadius is the radius of a one_ or sw12 (main line) line in µm (0 for straight)
	SwitchLength int64
	// SwitchLength is the length of a sw12's swithcing side line in µm
	SwitchRadius int64
	// SwitchRadius is the radius of a sw12's swithcing side line in µm (0 for straight)
	Conns []LineConn
	// Conns is the connections on a line, in order dependent on type.
}

/*
func NewLineKatoS248(conns []LineConn) Line {
	return Line{
		Type:      LineTypeOne,
		OneLength: 248e3,
		OneRadius: 0,
		Conns:     conns,
	}
}

func NewLineKatoR315D45(conns []LineConn) Line {
	return Line{
		Type:      LineTypeOne,
		OneLength: 494801, // TODO: choose a number that works well (fits well, etc)
		OneRadius: 315e3,
		Conns:     conns,
	}
}
*/

type LineType [4]rune

/*
var (
	LineTypeOne LineType = [4]rune("one_")
	// LineTypeOne is a "normal" line that does not converge nor diverge from other lines.
	// Connections: 2
	LineTypeSwitch12 LineType = [4]rune("sw12")
	// LineTypeSwitch12 is a switch that splits one line into two. (e.g. Y point)
	// Connections: 3
	LineTypeSwitch22 LineType = [4]rune("sw22")
	// LineTypeSwitch22 is a switch that splits two lines into two. (e.g. double crossover)
	// Connections: 4
)
*/

// LineConn connects a line to another line.
type LineConn struct {
	Other LineID
}
