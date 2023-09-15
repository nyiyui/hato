package tal

import (
	"github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	. "nyiyui.ca/hato/sakayukari"
)

func (g *Guide) View(model ActorRef) Actor {
	a := Actor{
		Comment: "tal-view",
		InputCh: make(chan Diffuse1),
		Inputs:  []ActorRef{model},
		Type: ActorType{
			Input: true,
		},
	}
	state := widgets.NewParagraph()
	state.Text = "tal-view"
	state.SetRect(0, 6, 30, 5)
	termui.Render(state)
	go func() {
	}()
	return a
}
