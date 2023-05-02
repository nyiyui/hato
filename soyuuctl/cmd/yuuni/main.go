package main

import (
	"log"

	"nyiyui.ca/soyuu/soyuuctl/yuuni"
)

func main() {
	err := yuuni.Main()
	if err != nil {
		log.Fatal(err)
	}
}
