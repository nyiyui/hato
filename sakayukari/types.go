package sakayukari

import "fmt"

// ActorRef refernces a single Actor.
// Only positive numbers are valid, and negative numbers are reserved.
// A 32-bit number was chosen as some (most?) MCUs do not have 64-bit numbers
type ActorRef struct {
	// Node  int32
	Index int
}

// Loopback, when set as the Origin in Diffuse1, makes the diffuse send to itself.
var Loopback = ActorRef{Index: -2}

// Publish, when set as the Origin in Diffuse1, sends the diffuse to all its dependents.
var Publish = ActorRef{Index: -3}

func (r ActorRef) String() string {
	return fmt.Sprintf("<a:%x>", r.Index)
}

// Diffuse1 is a change of an actor's output value.
type Diffuse1 struct {
	// Origin of this diffusion,
	Origin ActorRef

	// Value is the new value.
	Value Value
}

// Value is any value used by actors.
// It is fmt.Stringer for debugging purposes.
type Value interface {
	fmt.Stringer
	// TODO: add more methods for e.g. debugging?
}

// Actor is a single asynchronous actor.
// This can be used to emulate e.g. linear actors.
type Actor struct {
	// Comment is a human-friendly description about this actor.
	Comment string

	// InputCh is all diffuses relevant; this is asynchronous, meaning it can be sent at any time. It must be received within a reasonable time, as the runtime may rely on this completing to un-freeze the main loop.
	InputCh chan Diffuse1

	// OutputCh is all diffuses from the Actor; this is asynchronous, meaning it can be sent at any time.
	OutputCh chan Diffuse1

	// Inputs, has ActorRef to all actors it depends on.
	Inputs []ActorRef

	// Outputs, if not nil, has ActorRef to all actors it can change. If nil, it can change any actor.
	Outputs []ActorRef

	// // GetValue gets the latest value of this actor.
	// // This must terminate within a reasonable amount of time.
	// // The return value of GetValue does not change until the next Output is received.
	// GetValue func() Value

	Type ActorType
}

type ActorType struct {
	Input  bool
	Output bool
	// LinearInput (NOT IMPLEMENTED YET!) ensures the actor's InputCh are input linearly.
	LinearInput bool
}

type Graph struct {
	Actors []Actor
}
