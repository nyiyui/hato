package main

import (
	"time"

	"nyiyui.ca/hato/sakayukari/audio"
)

func main() {
	for range time.NewTicker(800 * time.Millisecond).C {
		audio.Play()
	}
}
