package layout

import (
	"math"

	"nyiyui.ca/hato/sakayukari/conn"
)

func InitTestbench1() (*Layout, error) {
	breadboard := func(line string) LineID {
		return LineID{
			Conn: conn.Id{"soyuu-line", "v1", "1"},
			Line: line,
		}
	}
	newBoard := func(line string) LineID {
		return LineID{
			Conn: conn.Id{"soyuu-line", "v2", "1"},
			Line: line,
		}
	}
	y, err := Connect([]Line{
		Line{
			Comment:   "1/D",
			PortB:     Port{Length: 312000},
			PowerConn: breadboard("D"),
		},
		Line{
			Comment:   "2/C",
			PortB:     Port{Length: 312000},
			PowerConn: breadboard("C"),
		},
		Line{
			Comment:   "3/B",
			PortB:     Port{Length: 312000},
			PowerConn: breadboard("B"),
		},
		Line{
			Comment:   "4/A",
			PortB:     Port{Length: 312000},
			PowerConn: breadboard("A"),
		},
		Line{
			Comment:   "5",
			PortB:     Port{Length: 312000},
			PowerConn: newBoard("A"),
		},
	})
	return &y, err
}

func InitTestbench2() (*Layout, error) {
	breadboard := func(line string) LineID {
		return LineID{
			Conn: conn.Id{"soyuu-line", "v2", "4"},
			Line: line,
		}
	}
	newBoard := func(line string) LineID {
		return LineID{
			Conn: conn.Id{"soyuu-line", "v2", "2"},
			Line: line,
		}
	}
	_ = newBoard
	length := 560000 + 718000*math.Pi*2*15/360
	y, err := Connect([]Line{
		//Line{
		//	Comment:   "Z",
		//	PortB:     Port{Length: 188000},
		//	PowerConn: newBoard("A"),
		//},
		Line{
			Comment: "Y",
			PortB:   Port{Length: 746000},
			PortC: Port{Length: uint32(length), ConnInline: []Line{
				Line{
					Comment:   "W",
					PortB:     Port{Length: 560000},
					PowerConn: breadboard("B"),
				},
			}},
			PowerConn:  breadboard("A"),
			SwitchConn: breadboard("C"),
		},
		Line{
			Comment:   "X",
			PortB:     Port{Length: 560000},
			PowerConn: breadboard("D"),
		},
	})
	return &y, err
}
