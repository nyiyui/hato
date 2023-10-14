// Package kato contains preset lengths for the KATO Unitrack series of model railroad tracks.
package kato

import "math"

const (
	r718_15 = (2 * math.Pi * 718_000) * (15.0 / 360)
	r481_15 = (2 * math.Pi * 481_000) * (15.0 / 360)
)
const (
	R718_15 = 187972
	R481_15 = 125926
	// EP481_15S is the straight side of a EP481-15L/R switch track.
	EP481_15S uint32 = 126_000
	// S60 is commonly found in EP481 sets.
	S60 uint32 = 60_000
	// S62 is commonly found in EP481 sets.
	S62 uint32 = 62_000
	// S62F is the common feeeder track (product #20-041)
	S62F        = S62
	S64  uint32 = 64_000
	S248 uint32 = 248_000
	S124 uint32 = 124_000
)
