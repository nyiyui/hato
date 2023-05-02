package sakayukari

import (
	"errors"
	"fmt"
	"log"
	"reflect"
)

var ErrShutdown = shutdownError{}

type shutdownError struct{}

func (s shutdownError) Error() string {
	return "shutdown"
}
func (s shutdownError) String() string {
	return "shutdown"
}

type Value interface {
	fmt.Stringer
}

type ActorIndex = int

type IndexAndValue struct {
	Index ActorIndex
	Value Value
}

type GraphState struct {
	States []Value
}

type Actor struct {
	DependsOn   []ActorIndex
	UpdateFunc  func(self *Actor, gs GraphState) (updated Value)
	RecvChan    chan Value
	SideEffects bool
	Comment     string
}

type Graph struct {
	Actors  []Actor
	DeptsOf [][]ActorIndex
	// DeptsOf has the dependents of Actors.
	// DeptsOf[i] is the list of actors that have actor i in the Actor.DependsOn fields.
	State   GraphState
	updates chan IndexAndValue
}

func NewGraph(actors []Actor) *Graph {
	g := new(Graph)
	g.Actors = actors
	g.State = GraphState{
		States: make([]Value, len(g.Actors)),
	}
	g.updates = make(chan IndexAndValue)
	g.calcDeptsOf()
	return g
}

func (a Actor) check() error {
	if !a.SideEffects && a.RecvChan != nil {
		return errors.New("pure actors must not have RecvChan")
	}
	return nil
}

func (g *Graph) calcDeptsOf() {
	dependents := make([][]ActorIndex, len(g.Actors))
	for i, actor := range g.Actors {
		for _, k := range actor.DependsOn {
			dependents[k] = append(dependents[k], i)
		}
	}
	g.DeptsOf = dependents
}

func (g *Graph) Check() error { return g.check() }

func (g *Graph) check() error {
	if g.DeptsOf == nil {
		panic("Graph.DeptsOf not calculated")
	}
	for i, actor := range g.Actors {
		if !actor.SideEffects && len(g.DeptsOf[i]) == 0 {
			return UselessActorError{I: i}
		}
		if err := actor.check(); err != nil {
			return fmt.Errorf("actor i%d: %w", i, err)
		}
	}
	return nil
}

func (g *Graph) Exec() {
	if g.DeptsOf == nil {
		panic("Graph.DeptsOf not calculated")
	}

	// setup cases
	cases := []reflect.SelectCase{
		reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(g.updates),
		},
	}
	caseIs := []int{-1}
	for i, actor := range g.Actors {
		if actor.SideEffects && actor.RecvChan != nil {
			cases = append(cases, reflect.SelectCase{
				Dir:  reflect.SelectRecv,
				Chan: reflect.ValueOf(actor.RecvChan),
			})
			caseIs = append(caseIs, i)
		}
	}
	if len(cases) != len(caseIs) {
		panic("len(cases) != len(caseIs)")
	}
	log.Printf("listening for %+v", caseIs)

	// select on all side-effctive actors
ActorLoop:
	for {
		chosen, recv, recvOK := reflect.Select(cases)
		if !recvOK {
			panic("recvOK is false but only SelectRecv is used")
		}
		i := caseIs[chosen]
		if chosen == 0 {
			iav := recv.Interface().(IndexAndValue)
			i = iav.Index
			g.State.States[i] = iav.Value
		} else {
			g.State.States[i] = recv.Interface().(Value)
		}
		//log.Printf("loop for index %d %s", i, g.Actors[i].Comment)
		depts := g.DeptsOf[i]
		updated := make([]bool, len(g.Actors))
		for len(depts) > 0 {
			j := depts[0]
			actor := g.Actors[j]
			// TODO: hide values in g.State that are not relied upon by actor
			// TODO: confine actors with side-effects runtime (prevent hanging)
			if !updated[j] {
				val := actor.UpdateFunc(&actor, g.State)
				if _, ok := val.(shutdownError); ok {
					log.Printf("shutdown due to actor i%d", j)
					break ActorLoop
				}
				g.State.States[j] = val
				updated[j] = true
			}
			depts = append(depts[1:], g.DeptsOf[j]...)
		}
	}
}

type UselessActorError struct {
	I int
}

func (u UselessActorError) Error() string {
	return fmt.Sprintf("useless actor i%d", u.I)
}
