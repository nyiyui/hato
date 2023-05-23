package main

import (
	"log"
	"os"

	"nyiyui.ca/hato/sakayukari/ctl"
)

func main() {
	err := ctl.Main()
	if err != nil {
		log.Print(err)
		os.Exit(3)
	}
}
