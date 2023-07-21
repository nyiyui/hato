package runtime

/*

import (
	"encoding/base64"
	"encoding/json"
)

// Replay sends the following values to the runtime.
// If SerializedValue.Time is present in all values, they are sent at same time intervals between values.
// If SerializedValue.Time is not present in all values, they are sent in the same order at unknown intervals.
// If SerializedValue.Time is present in some but not all values, this function panics.
func (i *Instance) Replay(values []SerializedValue) {
	panic("not implemented yet")
	timeCount := 0
	for _, value := range values {
		if values.Time {
			timeCount++
		}
	}
	if timeCount != 0 && timeCount != len(values) {
		panic("SerializedValue.Time is present in some but not all values")
	}
	for i, sv := range values {
		data, err := base64.StdEncoding.DecodeString(sv.Value)
		if err != nil {
			t.Fatalf("value: base64 decode: %s", err)
		}
		var d Diffuse1
		if err := json.Unmarshal(data, &d); err != nil {
			t.Fatalf("value: json decode: %s", err)
		}
	}
}
*/
