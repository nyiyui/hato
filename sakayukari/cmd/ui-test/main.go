package main

import (
	"log"
	"os"

	"nyiyui.ca/hato/sakayukari/ui"
)

func main() {
	err := ui.Main()
	if err != nil {
		log.Print(err)
		os.Exit(3)
	}
}
