package model

import (
	_ "embed"
	"encoding/json"
)

//go:embed 5car-map.json
var timingMapRaw []byte

var timingMap = map[int]int64{}

func init() {
	json.Unmarshal(timingMapRaw, timingMap)
}
