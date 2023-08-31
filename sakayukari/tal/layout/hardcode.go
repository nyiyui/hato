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
			PortA:   Port{Direction: true},
			PortB:   Port{Length: 746000, Direction: false},
			PortC: Port{Length: uint32(length), Direction: false, ConnInline: []Line{
				Line{
					Comment:   "W",
					PortA:     Port{Direction: true},
					PortB:     Port{Length: 560000, Direction: false},
					PowerConn: breadboard("B"),
				},
			}},
			PowerConn:  breadboard("A"),
			SwitchConn: breadboard("C"),
		},
		Line{
			Comment:   "X",
			PortA:     Port{Direction: true},
			PortB:     Port{Length: 560000, Direction: false},
			PowerConn: breadboard("D"),
		},
	})
	return &y, err
}

func InitTestbench3() (*Layout, error) {
	yellow := func(line string) LineID {
		return LineID{
			Conn: conn.Id{"soyuu-line", "v2", "yellow"},
			Line: line,
		}
	}
	white := func(line string) LineID {
		return LineID{
			Conn: conn.Id{"soyuu-line", "v2", "white"},
			Line: line,
		}
	}
	_, _ = yellow, white
	normal := 282000*math.Pi*2*90/360 + 186000
	reverse := 282000*math.Pi*2*90/360 + 718000*math.Pi*2*15/360
	station := 2*64000 + 2*718000*math.Pi*2*15/360 + 248000
	_, _, _ = normal, reverse, station
	y, err := Connect([]Line{
		//Line{
		//	Comment:   "Z",
		//	PortA:     Port{Direction: true},
		//	PortB:     Port{Length: 128000, Direction: false},
		//	PowerConn: yellow("A"),
		//},
		//Line{
		//	Comment:   "Y",
		//	PortB:     Port{Length: 3*248000 + 128000, Direction: true},
		//	PowerConn: yellow("B"),
		//},
		//Line{
		//	Comment: "X",
		//	PortA:   Port{Direction: true},
		//	PortB:   Port{Length: uint32(normal), Direction: false},
		//	PortC: Port{Length: uint32(reverse), Direction: false, ConnInline: []Line{
		//		Line{
		//			Comment:   "V",
		//			PortA:     Port{Direction: false},
		//			PortB:     Port{Length: uint32(station), Direction: true},
		//			PowerConn: white("A"),
		//		},
		//	}},
		//	PowerConn:  yellow("C"),
		//	SwitchConn: white("B"),
		//},
		//Line{
		//	Comment:   "W",
		//	PortA:     Port{Direction: true},
		//	PortB:     Port{Length: 248000 * 3, Direction: false},
		//	PowerConn: white("C"),
		//},
		Line{
			Comment:   "Z",
			PortA:     Port{Direction: true},
			PortB:     Port{Length: 128000, Direction: false},
			PowerConn: yellow("A"),
		},
		Line{
			Comment:   "Y",
			PortB:     Port{Length: 3*248000 + 128000, Direction: true},
			PowerConn: yellow("B"),
		},
		Line{
			Comment: "X",
			PortA:   Port{Direction: true},
			PortB:   Port{Length: uint32(normal), Direction: false},
			PortC: Port{Length: uint32(reverse), Direction: false, ConnInline: []Line{
				Line{
					Comment:   "V",
					PortA:     Port{Direction: false},
					PortB:     Port{Length: uint32(station), Direction: true},
					PowerConn: white("A"),
				},
			}},
			PowerConn:  yellow("C"),
			SwitchConn: white("B"),
		},
		Line{
			Comment:   "W",
			PortA:     Port{Direction: false},
			PortB:     Port{Length: 248000 * 3, Direction: true},
			PowerConn: white("C"),
		},
	})
	return &y, err
}

func InitTestbench4() (*Layout, error) {
	board := func(line string) LineID {
		return LineID{
			Conn: conn.Id{"soyuu-line", "v2", "deepgreen"},
			Line: line,
		}
	}
	r183 := math.Pi * 183000 * 2
	y, err := Connect([]Line{
		Line{
			Comment:   "A",
			PortA:     Port{Direction: false},
			PortB:     Port{Length: 2 * 248000, Direction: true},
			PowerConn: board("A"),
		},
		Line{
			Comment:   "B",
			PortA:     Port{Direction: false},
			PortB:     Port{Length: 64000 + uint32(r183/2), Direction: true},
			PowerConn: board("B"),
		},
		Line{
			Comment:   "C",
			PortA:     Port{Direction: true},
			PortB:     Port{Length: uint32(r183 / 2), Direction: false},
			PowerConn: board("C"),
		},
	})
	return &y, err
}
