package sakayukari

import (
	"fmt"
	"log"
	"testing"
)

type valueString string

func (v valueString) String() string { return string(v) }

func TestGraph(t *testing.T) {
	ch1 := make(chan Value, 3)
	ch1 <- valueString("Jason")
	ch1 <- valueString("Patrick")
	ch1 <- valueString("Misheel")
	var g *Graph
	g = NewGraph([]Actor{
		Actor{
			RecvChan:    ch1,
			SideEffects: true,
		},
		Actor{
			DependsOn: []ActorIndex{0},
			UpdateFunc: func(self *Actor, gs GraphState) Value {
				return valueString(fmt.Sprintf("Hello, %s!", gs.States[0]))
			},
		},
		Actor{
			DependsOn: []ActorIndex{1},
			UpdateFunc: func(self *Actor, gs GraphState) Value {
				log.Printf("updated: %s", gs.States[1])
				if gs.States[1].String() == "Hello, Patrick!" {
					log.Print("request shutdown")
					return ErrShutdown
				}
				return nil
			},
			SideEffects: true,
		},
	})
	err := g.check()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("checked")
	g.Exec()
}
