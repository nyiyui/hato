package main

import (
	"log"
	"os"

	"nyiyui.ca/hato/sakayukari/ctl"

	"net/http"
	_ "net/http/pprof"
)

func main() {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	err := ctl.Main()
	if err != nil {
		log.Print(err)
		os.Exit(3)
	}
}
