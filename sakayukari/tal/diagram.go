// Diagram - timetable for trains
package tal

import . "nyiyui.ca/hato/sakayukari"

type DiagramConf struct {
	Guide ActorRef
}

type diagram struct {
	conf  DiagramConf
	actor *Actor
}

type Schedule struct {
	TSs []TrainSchedule
}

type TrainSchedule struct {
	Segments []Segment
}

type Position struct {
	Line LineID
	// Precise is the position from port A in µm.
	Precise uint32
}

type Segment struct {
	Time uint64
	// Target is the target position for the train to go to by the time (above).
	// Note: the first Segment lists the starting position of the train (it is unspecified what Diagram will do if the train is not near (on the same Lines) that position).
	Target Position
	Speed  uint32
}

func Diagram(conf DiagramConf) *Actor {
	a := &Actor{
		Comment:  "tal-diagram",
		InputCh:  make(chan Diffuse1),
		OutputCh: make(chan Diffuse1),
		Inputs:   make([]ActorRef, 0),
		Type: ActorType{
			Input:       true,
			LinearInput: true,
			Output:      true,
		},
	}
	d := &diagram{
		conf:  conf,
		actor: a,
	}
	go d.loop()
	return a
}

func (d *diagram) loop() {
}
