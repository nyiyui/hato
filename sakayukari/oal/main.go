package oal

import "nyiyui.ca/hato/sakayukari/tal/layout"

type AllStatus struct {
	Operations []Operation
}

type Station struct {
	DefaultPosition layout.Position
}

type Operation struct {
	TrainI       int
	GoalStationI int
	GoalPosition layout.Position
	Extra        map[string]string
	Status       OperationStatus
}

type OperationStatus struct {
}
