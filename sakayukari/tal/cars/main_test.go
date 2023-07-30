package cars

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

//go:embed test.json
var testJson []byte

func TestCarsJSON(t *testing.T) {
	var data Data
	err := json.Unmarshal(testJson, &data)
	if err != nil {
		t.Fatalf("unmarshal: %s", err)
	}
	testJson2, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("unmarshal: %s", err)
	}
	if bytes.Equal(testJson, testJson2) {
		t.Fatal("mismatch")
	}
}

func TestTrailerLength(t *testing.T) {
	var data Data
	err := json.Unmarshal(testJson, &data)
	if err != nil {
		t.Fatalf("unmarshal: %s", err)
	}
	f := data.Forms[uuid.MustParse("2fe1cbb0-b584-45f5-96ec-a9bfd55b1e91")]
	t.Logf("form: %#v", f)
	sideA, sideB := f.TrailerLength()
	if sideA != 131000 {
		t.Fatalf("wrong sideA")
	}
	if sideB != 131000 {
		t.Fatalf("wrong sideB")
	}
}
