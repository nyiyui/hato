package layout

import "nyiyui.ca/hato/sakayukari/conn"

func InitTestbench() (*Layout, error) {
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
			Comment:   "1",
			PortB:     Port{Length: 312000},
			PowerConn: breadboard("D"),
		},
		Line{
			Comment:   "2",
			PortB:     Port{Length: 312000},
			PowerConn: breadboard("C"),
		},
		Line{
			Comment:   "3",
			PortB:     Port{Length: 312000},
			PowerConn: breadboard("B"),
		},
		Line{
			Comment:   "4",
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
