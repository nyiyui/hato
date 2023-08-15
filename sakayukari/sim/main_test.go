package sim

import (
	"encoding/json"
	"testing"

	_ "embed"

	"nyiyui.ca/hato/sakayukari/tal"
	"nyiyui.ca/hato/sakayukari/tal/cars"
	"nyiyui.ca/hato/sakayukari/tal/layout"
)

//go:embed cars.json
var carsJSON []byte

func TestMain(t *testing.T) {
	y, err := layout.InitTestbench3()
	if err != nil {
		t.Fatal(err)
	}
	var carsData cars.Data
	{
		err = json.Unmarshal(carsJSON, &carsData)
		if err != nil {
			t.Fatalf("parse cars.json: %s", err)
		}
	}
	s := New(SimulationConf{
		Layout: y,
		ModelConf: tal.ModelConf{
			Cars: carsData,
			RFIDs: []tal.RFID{
				{Position: layout.Position{
					LineI:   y.MustLookupIndex("Y"),
					Precise: 252000,
					Port:    layout.PortB,
				}},
				{Position: layout.Position{
					LineI:   y.MustLookupIndex("X"),
					Precise: 50000,
					Port:    layout.PortB,
				}},
			},
		},
	})
	s.Run()
}
