package yuuni

import (
	"fmt"
	"sort"
	"strings"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"nyiyui.ca/soyuu/soyuuctl/conn"
	"nyiyui.ca/soyuu/soyuuctl/sakayukari"
)

func breakbeam(show map[string]string) sakayukari.Actor2 {
	state := widgets.NewParagraph()
	state.Text = "loading"
	state.SetRect(0, 0, 60, 27)
	ui.Render(state)
	keys := make([]string, len(show))
	for _, key := range show {
		keys = append(keys, key)
	}
	labels := make([]string, 0, len(show))
	for label := range show {
		labels = append(labels, label)
	}
	sort.Strings(labels)
	return sakayukari.Actor2{
		DependsOn: keys,
		UpdateFunc: func(self *sakayukari.Actor, gsm sakayukari.GraphStateMap, gs sakayukari.GraphState) (updated sakayukari.Value) {
			b := new(strings.Builder)
			for _, label := range labels {
				key := show[label]
				val := gs.States[gsm[key]]
				if val == nil {
					fmt.Fprintf(b, "%s: ?\n", label)
					continue
				}
				//log.Printf("render %s %#v", key, gs.States[gsm[key]])
				switch val := val.(type) {
				case conn.ValSeen:
					if val.Seen {
						fmt.Fprintf(b, "%s: +\t%d\n", label, val.Monotonic)
					} else {
						fmt.Fprintf(b, "%s: -\t%d\n", label, val.Monotonic)
					}
				default:
					fmt.Fprintf(b, "%s: %v", label, val)
				}
			}
			state.Text = b.String()
			ui.Render(state)
			return nil
		},
		SideEffects: true,
		Comment:     fmt.Sprintf("breakbeam %s", keys),
	}
}
