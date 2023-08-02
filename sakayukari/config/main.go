package config

import (
	"nyiyui.ca/hato/sakayukari/conn"
	"nyiyui.ca/hato/sakayukari/tal/cars"
)

type Config struct {
	Lines []Line    `json:"lines"`
	RFIDs []RFID    `json:"rfids"`
	Cars  cars.Data `json:"cars"`
}

type Line struct {
	ConnID  conn.Id `json:"conn-id"`
	SubLine string  `json:"sub-line"`
}

type RFID struct {
	ConnID   conn.Id  `json:"conn-id"`
	Position Position `json:"position"`
}

type Position struct {
	Line    string `json:"line"`
	Precise int64  `json:"precise"`
	Port    string `json:"port"`
}
