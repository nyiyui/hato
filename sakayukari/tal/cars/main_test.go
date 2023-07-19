package cars

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"testing"
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
