package cars

type CarSet struct {
	// Length of the whole carset in µm.
	// This may not be the sum of the car's individual lengths due to couplers, etc.
	Length uint32
	// Cars is the list of cars in this carset.
	// They must be ordered so that side A of a car is adjacent to side B of the next car (excluding the first and last cars).
	// Side A of the first car, and side B of the last car is not adjacent to any other car in this carset.
	Cars []Car
}

type Car struct {
	// Length of the car in µm.
	Length   uint32
	MifareID MifareID
	// MifarePosition is the position of the Mifare card/tag from side A to side B.
	MifarePosition uint32
}

// MifareID represents a 7-byte UID for a Mifare card/tag.
// The representation for a 4-byte NUID is unsupported (for now, I don't have those tags so I can't test them).
type MifareID [7]byte
