package main

import (
	"fmt"

	"nyiyui.ca/hato/sakayukari/tal/layout"
)

func main() {
	y, err := layout.InitTestbench6()
	if err != nil {
		panic(err)
	}
	path := y.MustFullPathTo(layout.LinePort{
		LineI: y.MustLookupIndex("nagase1"),
		PortI: layout.PortA,
	}, layout.LinePort{
		LineI: y.MustLookupIndex("mitouc2"),
		PortI: layout.PortB,
	})
	pos, err := y.OffsetToPosition(path, 0)
	if err != nil {
		panic(err)
	}
	fmt.Printf("offset 0 → position %#v\n", pos)
	pos, err = y.OffsetToPosition(path, (248+248+62+186+1)*1000)
	if err != nil {
		panic(err)
	}
	fmt.Printf("offset nagase1 + 1mm → position %#v\n", pos)
}
