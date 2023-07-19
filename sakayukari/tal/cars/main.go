package cars

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

type Data struct {
	Forms map[uuid.UUID]Form `json:"sets"` // json struct tag isn't actually used but kept for docs purposes
}

type dataJSON struct {
	Sets map[string]Form `json:"sets"`
}

func (d Data) MarshalJSON() ([]byte, error) {
	d3 := dataJSON{Sets: map[string]Form{}}
	for key, cs := range d.Forms {
		d3.Sets[key.String()] = cs
	}
	return json.Marshal(d3)
}

func (d *Data) UnmarshalJSON(data []byte) error {
	var d3 dataJSON
	err := json.Unmarshal(data, &d3)
	if err != nil {
		return err
	}
	d2 := Data{Forms: map[uuid.UUID]Form{}}
	for key, cs := range d3.Sets {
		u2, err := uuid.Parse(key)
		if err != nil {
			return fmt.Errorf("key %s: parse key as UUID: %w", key, err)
		}
		d2.Forms[u2] = cs
	}
	*d = d2
	return nil
}

type FormCarI struct {
	Form  uuid.UUID
	Index int
}

// Form represents a single formation.
type Form struct {
	Comment string `json:"comment"`
	// Length of the whole carset in µm.
	// This may not be the sum of the car's individual lengths due to couplers, etc.
	Length uint32 `json:"length"`
	// Cars is the list of cars in this carset.
	// They must be ordered so that side A of a car is adjacent to side B of the next car (excluding the first and last cars).
	// Side A of the first car, and side B of the last car is not adjacent to any other car in this carset.
	Cars []Car `json:"cars"`
}

type Car struct {
	Comment string `json:"comment"`
	// Length of the car in µm.
	Length   uint32   `json:"length"`
	MifareID MifareID `json:"mifare-id"`
	// MifarePosition is the position of the Mifare card/tag from side A to side B.
	MifarePosition uint32 `json:"mifare-pos"`
}

// MifareID represents a 7-byte UID for a Mifare card/tag.
// The representation for a 4-byte NUID is unsupported (for now, I don't have those tags so I can't test them).
type MifareID [7]byte

func (m *MifareID) UnmarshalJSON(data []byte) error {
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return err
	}
	if len(b) != 7 {
		return errors.New("Mifare ID length must be 7")
	}
	*m = MifareID(*(*[7]byte)(b))
	return nil
}

func (m *MifareID) MarshalJSON() ([]byte, error) {
	return json.Marshal(hex.EncodeToString(m[:]))
}
