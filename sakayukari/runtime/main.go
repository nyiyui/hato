package runtime

import (
	"fmt"
	"log"
	"os"
	"reflect"
	"runtime/pprof"
	"sync"
	"time"

	. "nyiyui.ca/hato/sakayukari"
)

type Instance struct {
	g           *Graph
	traceOutput *os.File
	traceLock   sync.Mutex
}

func NewInstance(g *Graph) *Instance {
	return &Instance{g: g}
}

func (i *Instance) ReplaceActor(ref ActorRef, a Actor) {
	panic("not implemented")
}

// TODO: change Graph while reducing execution interruption

func removeDuplicate[T comparable](sliceList []T) []T {
	allKeys := make(map[T]bool)
	list := []T{}
	for _, item := range sliceList {
		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			list = append(list, item)
		}
	}
	return list
}

func (i *Instance) dependsOn() [][]int {
	dependsOn := make([][]int, len(i.g.Actors))
	for j, actor := range i.g.Actors {
		for _, k := range actor.Inputs {
			dependsOn[k.Index] = append(dependsOn[k.Index], j)
		}
	}
	// deduplication is needed to prevent sending the same Diffuse1 multiple times to an Actor
	for j, actor := range i.g.Actors {
		dependsOn[j] = removeDuplicate(dependsOn[j])
		_ = actor
		// log.Printf("--- INPUTS %d: %#v", j, actor.Inputs)
	}
	return dependsOn
}

func (i *Instance) Check() error {
	err := i.initRecord()
	if err != nil {
		log.Printf("initRecord: %s", err)
	}
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
	if i.traceOutput != nil {
		defer func() {
			err := i.traceOutput.Close()
			if err != nil {
				log.Printf("sakayukari-runtime: close traceOutput: %s", err)
			}
		}()
	}
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
		// log.Printf("WAITING")
		chosen, recv, recvOK := reflect.Select(cases)
		if !recvOK {
			panic("recvOK is false but only SelectRecv is used")
		}
		go func() {
			var caseI int
			caseI = caseIs[chosen]
			d := recv.Interface().(Diffuse1)
			//log.Printf("got: %s", d)
			if d.Origin == (ActorRef{}) || d.Origin == Publish {
				// self if blank
				d.Origin = ActorRef{Index: caseI}
				// only do dependencies if the actor itself publishes a new value; if the actor sends it to a different actor, that actor can decide to publichs a new value or not
				// log.Printf("sending to deps of %s: %#v", d.Origin, dependsOn[d.Origin.Index])
				i.record(&d, dependsOn[d.Origin.Index])
				for _, j := range dependsOn[d.Origin.Index] {
					i.send(j, &d)
				}
			} else if d.Origin == Loopback {
				// send to self
				//d.Origin = ActorRef{Index: caseI}
				i.record(&d, []int{caseI})
				// TODO: do we want to record if this was a loopback or not too?
				i.send(caseI, &d)
			} else {
				// if not self, this Diffuse1 is a set to another actor
				dest := d.Origin.Index
				state[dest] = d.Value
				origin := i.g.Actors[dest]
				if !origin.Type.Input {
					panic(fmt.Sprintf("input to non-input actor %s %s", d.Origin, origin.Comment))
				}
				d.Origin.Index = caseI
				//log.Printf("send to %s: %s", d.Origin, d)
				i.record(&d, []int{dest})
				i.send(dest, &d)
				//log.Printf("sent to %s: %s", d.Origin, d)
			}
			//log.Print("done this loop")
			// TODO: handle hanging actors
		}()
	}
}

func (i *Instance) send(actorI int, d *Diffuse1) {
	//i.g.Actors[actorI].InputCh <- *d
	select {
	case i.g.Actors[actorI].InputCh <- *d:
	case <-time.After(200 * time.Millisecond):
		pprof.Lookup("goroutine").WriteTo(os.Stderr, 1)
		panic(fmt.Sprintf("actor %s %s timed out: %#v", ActorRef{Index: actorI}, i.g.Actors[actorI].Comment, d))
	}
}
