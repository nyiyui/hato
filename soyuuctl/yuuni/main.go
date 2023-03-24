package yuuni

import (
	"fmt"
	"log"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"nyiyui.ca/soyuu/soyuuctl/conn"
)

type lineState struct {
	power     uint8
	direction bool
}

type Yuuni struct {
	s *conn.State

	state *widgets.Paragraph

	lineState  *widgets.Paragraph
	lineStates map[string]*lineState
}

func (y *Yuuni) stHook(name string, v conn.Val) {
	switch v := v.(type) {
	case conn.ValAttitude:
		switch v.State {
		case conn.STStateSide:
			y.state.Text = fmt.Sprintf("side\n%d um\n%d um/s\nts %d", v.Position, v.Velocity, v.Monotonic)
		case conn.STStateTop:
			y.state.Text = "top"
		case conn.STStateBase:
			y.state.Text = "base"
		}
		ui.Render(y.state)
	}
}

func newYuuni() (*Yuuni, error) {
	s := conn.NewState()
	err := s.Find()
	if err != nil {
		log.Fatalf("find: %s", err)
	}
	state := widgets.NewParagraph()
	state.Text = "no state"
	state.SetRect(25, 0, 40, 5)
	ls := widgets.NewParagraph()
	ls.Text = "no line state"
	ls.SetRect(40, 0, 60, 5)
	return &Yuuni{
		s:         s,
		state:     state,
		lineState: ls,
		lineStates: map[string]*lineState{
			"A": new(lineState),
			"B": new(lineState),
		},
	}, nil
}

func (y *Yuuni) allStop() {
	y.s.LineReq("A", conn.ReqLine{
		Brake:     false,
		Direction: true,
		Power:     00,
	})
	y.s.LineReq("B", conn.ReqLine{
		Brake:     false,
		Direction: true,
		Power:     00,
	})
}

func Main() error {
	y, err := newYuuni()
	if err != nil {
		return err
	}
	var c *conn.Conn
	ok := false
	for !ok {
		c, ok = y.s.GetST("0")
	}
	func() {
		c.HooksLock.Lock()
		defer c.HooksLock.Unlock()
		c.Hooks = append(c.Hooks, func(v conn.Val) { y.stHook("0", v) })
	}()
	y.allStop()

	if err := ui.Init(); err != nil {
		return fmt.Errorf("init termui: %w", err)
	}
	defer ui.Close()
	p := widgets.NewParagraph()
	p.Text = "soyuuctl-yuuni"
	p.SetRect(0, 0, 25, 5)
	ui.Render(p)
	ui.Render(y.state)
	for e := range ui.PollEvents() {
		changed := true
		switch e.ID {
		case "<C-c>":
			return nil
		case "0":
			go y.allStop()
			changed = false
		case "q":
			if y.lineStates["A"].power != 255 {
				y.lineStates["A"].power += 3
			}
		case "a":
			if y.lineStates["A"].power != 0 {
				y.lineStates["A"].power -= 3
			}
		case "z":
			y.lineStates["A"].direction = !y.lineStates["A"].direction
		case "w":
			if y.lineStates["B"].power != 255 {
				y.lineStates["B"].power += 3
			}
		case "s":
			if y.lineStates["B"].power != 0 {
				y.lineStates["B"].power -= 3
			}
		case "x":
			y.lineStates["B"].direction = !y.lineStates["B"].direction
		}
		if changed {
			y.s.LineReq("A", conn.ReqLine{
				Brake:     false,
				Direction: y.lineStates["A"].direction,
				Power:     y.lineStates["A"].power,
			})
			y.s.LineReq("B", conn.ReqLine{
				Brake:     false,
				Direction: y.lineStates["B"].direction,
				Power:     y.lineStates["B"].power,
			})
			y.lineState.Text = fmt.Sprintf("A %t\t%d\n", y.lineStates["A"].direction, y.lineStates["A"].power)
			y.lineState.Text += fmt.Sprintf("B %t\t%d\n", y.lineStates["B"].direction, y.lineStates["B"].power)
			ui.Render(y.lineState)
		}
	}
	return nil
}
