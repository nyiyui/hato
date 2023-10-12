package main

import (
	"log"

	"github.com/openacid/slimarray/polyfit"
)

func main() {
	fit := polyfit.NewFit(nil, nil, 2)
	fit.Add(0, 1)
	fit.Add(1, 1)
	fit.Add(2, 1)
	fit.Add(3, 1)
	//fit.Add(0, 1)
	//fit.Add(0, 1)
	//fit.Add(1, 3)
	//fit.Add(2, 7)
	//fit.Add(3, 13)
	coeffs := fit.Solve()
	log.Printf("coeffs: %#v", coeffs)
}
