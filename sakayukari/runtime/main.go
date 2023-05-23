package runtime

import (
	"fmt"
	"reflect"

	. "nyiyui.ca/hato/sakayukari"
)

type Instance struct {
	g *Graph
}

func NewInstance(g *Graph) *Instance {
	return &Instance{g}
}

func (i *Instance) ReplaceActor(ref ActorRef, a Actor) {
	panic("not implemented")
}

// TODO: change Graph while reducing execution interruption

func (i *Instance) dependsOn() [][]int {
	dependsOn := make([][]int, len(i.g.Actors))
	for j, actor := range i.g.Actors {
		for _, k := range actor.Inputs {
			dependsOn[k.Index] = append(dependsOn[k.Index], j)
		}
	}
	return dependsOn
}

func (i *Instance) Check() error {
	for i, actor := range i.g.Actors {
		if actor.Type.Output != (actor.OutputCh != nil) {
			return fmt.Errorf("actor %s %s: type mismatch: output", ActorRef{Index: i}, actor.Comment)
		}
		if actor.Type.Input != (actor.InputCh != nil) {
			return fmt.Errorf("actor %s %s: type mismatch: input", ActorRef{Index: i}, actor.Comment)
		}
		if actor.Type.Input == false && actor.Type.Output == false {
			return fmt.Errorf("actor %s %s: type: no i/o", ActorRef{Index: i}, actor.Comment)
		}
	}
	return nil
}

func (i *Instance) Diffuse() error {
	// setup cases
	cases := []reflect.SelectCase{}
	caseIs := []int{}
	for i, actor := range i.g.Actors {
		if !actor.Type.Output {
			continue
		}
		cases = append(cases, reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(actor.OutputCh),
		})
		caseIs = append(caseIs, i)
	}

	dependsOn := i.dependsOn()
	state := make([]Value, len(i.g.Actors))
	for {
		chosen, recv, recvOK := reflect.Select(cases)
		if !recvOK {
			panic("recvOK is false but only SelectRecv is used")
		}
		var caseI int
		caseI = caseIs[chosen]
		d := recv.Interface().(Diffuse1)
		if d.Origin == (ActorRef{}) {
			// self if blank
			d.Origin = ActorRef{Index: caseI}
		} else {
			// if not self, this Diffuse1 is a set to another actor
			state[d.Origin.Index] = d.Value
			origin := i.g.Actors[d.Origin.Index]
			if !origin.Type.Input {
				panic(fmt.Sprintf("input to non-input actor %s %s", d.Origin, origin.Comment))
			}
			origin.InputCh <- d
		}
		// log.Printf("%s dependsOn %#v", d.Origin, dependsOn[d.Origin.Index])
		for _, j := range dependsOn[d.Origin.Index] {
			dep := i.g.Actors[j]
			dep.InputCh <- d
		}
		// TODO: handle hanging actors
	}
}
