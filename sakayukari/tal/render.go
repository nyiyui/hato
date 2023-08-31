package tal

import (
	"fmt"
	"strings"

	"github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	. "nyiyui.ca/hato/sakayukari"
)

func GuideRender(guide ActorRef) Actor {
	a := Actor{
		Comment: "tal-guide-render",
		InputCh: make(chan Diffuse1),
		Inputs:  []ActorRef{guide},
		Type:    ActorType{Input: true},
	}
	go func() {
		state := widgets.NewParagraph()
		state.SetRect(0, 5, 70, 20)
		for diffuse := range a.InputCh {
			if diffuse.Origin != guide {
				continue
			}
			switch val := diffuse.Value.(type) {
			case GuideSnapshot:
				b := new(strings.Builder)
				for ti, t := range val.Trains {
					fmt.Fprintf(b, "%d %s\n", ti, &t)
					fmt.Fprintf(b, "%d %s\n", ti, t.Path)
					fmt.Fprintf(b, "%d %#v\n", ti, t)
				}
				state.Text = b.String()
				termui.Render(state)
			}
		}
	}()
	return a
}
