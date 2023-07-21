package tal

/*
import (
	_ "embed"

	. "nyiyui.ca/hato/sakayukari"
)

//go:embed hato-sakayukari-trace-record-2023-07-20-model
var trace []byte

func TestModelOnRecord(t *testing.T) {
	s := bufio.NewScanner(bytes.NewBuffer(trace))
	values := make([]SerializedValue, 0)
	for s.Scan() {
		if err := s.Err(); err != nil {
			t.Fatalf("scanner: %s", err)
		}
		var sv runtime.SerializedValue
		if err := json.Unmarshal(s.Bytes(), &sv); err != nil {
			t.Fatalf("json decode: %s", err)
		}
		if sv.Value == "" {
			continue
		}
		data, err := base64.StdEncoding.DecodeString(sv.Value)
		if err != nil {
			t.Fatalf("value: base64 decode: %s", err)
		}
		var d Diffuse1
		if err := json.Unmarshal(data, &d); err != nil {
			t.Fatalf("value: json decode: %s", err)
		}
		if d.Origin.Index != 4 { // didn't come from tal-guide
			continue
		}
		if strings.Contains(sv.Preview, "ReqLine") {
			continue
		}
		values = append(values, sv)
	}
}
*/
