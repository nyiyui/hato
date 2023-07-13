package tal

import (
	"bytes"
	"encoding/gob"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGobEncode(t *testing.T) {
	gs := GuideSnapshot{
		Trains: []Train{Train{
			Power:           70,
			noPowerSupplied: false,
			CurrentBack:     0,
			CurrentFront:    1,
			Path: []LinePort{
				{0, 1},
				{1, 1},
				{2, 1},
			},
			State: TrainStateNextAvail,
		}},
	}
	buf := new(bytes.Buffer)
	err := gob.NewEncoder(buf).Encode(gs)
	if err != nil {
		t.Errorf("encode: %s", err)
	}
	var res GuideSnapshot
	err = gob.NewDecoder(buf).Decode(&res)
	if err != nil {
		t.Errorf("decode: %s", err)
	}
	if !cmp.Equal(gs, res, cmp.AllowUnexported(Train{})) {
		t.Errorf("diff: %s", cmp.Diff(gs, res, cmp.AllowUnexported(Train{})))
	}
}
