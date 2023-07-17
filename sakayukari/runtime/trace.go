package runtime

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	. "nyiyui.ca/hato/sakayukari"
)

type serializedValue struct {
	Preview      string
	Value        []byte
	Destinations []int
}

// serialize tries to serialize as much as it can of v.
func serialize(v interface{}) (sv *serializedValue) {
	sv = new(serializedValue)
	sv.Preview = fmt.Sprintf("%#v", v)
	var err error
	buf := new(bytes.Buffer)
	err = json.NewEncoder(buf).Encode(v)
	if err == nil {
		sv.Value = buf.Bytes()
		return
	}
	return
}

func (i *Instance) initRecord() error {
	if i.traceOutput != nil {
		return errors.New("sakayukari-runtime: trace: init: already inited")
	}
	f, err := os.Create("/tmp/hato-sakayukari-trace-record")
	if err != nil {
		return err
	}
	i.traceOutput = f
	return nil
}

func (i *Instance) record(d *Diffuse1, dests []int) {
	if i.traceOutput == nil {
		return
	}
	i.traceLock.Lock()
	defer i.traceLock.Unlock()
	sv := serialize(d)
	sv.Destinations = dests
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(sv)
	if err != nil {
		log.Printf("sakayukari-runtime: trace: record (encode) %#v: %s", d, err)
		return
	}
	_, err = io.Copy(i.traceOutput, buf)
	if err != nil {
		log.Printf("sakayukari-runtime: trace: record (copy) %#v: %s", d, err)
		return
	}
}
