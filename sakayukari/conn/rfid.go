package conn

import (
	"bufio"
	"encoding/hex"
	"log"
	"strconv"
	"strings"
	"time"

	. "nyiyui.ca/hato/sakayukari"
)

type handlerRFID struct{}

func (_ handlerRFID) NewBlankActor() Actor {
	return Actor{
		Comment:  "blank handlerRFID",
		OutputCh: make(chan Diffuse1),
		Type: ActorType{
			Output: true,
		},
	}
}

func (_ handlerRFID) HandleConn(a Actor, c *Conn) {
	reader := bufio.NewReader(c.F)
	for {
		lineRaw, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("%s: read line: %s", c.Path, err)
			continue
		}
		if !strings.HasPrefix(lineRaw, " D") {
			continue
		}
		now := time.Now()
		line := lineRaw[2:]
		parts := strings.Split(strings.TrimSpace(line), " ")
		length := -1
		var data []byte
		for _, part := range parts {
			switch part[0] {
			case 'N':
			// NOTE: not implemented yet (not required as there aren't multiple readers per device for now)
			case 'L':
				length_, err := strconv.ParseUint(part[1:], 10, 32)
				if err != nil {
					log.Printf("length decoding failed: %s", err)
				}
				length = int(length_)
			case 'V':
				data, err = hex.DecodeString(part[1:])
				if err != nil {
					log.Printf("data (ID) decoding failed: %s", err)
				}
			}
		}
		if length != -1 && length != len(data) {
			log.Printf("data length mismatch: wanted %d, got %d", length, len(data))
		} else if length == -1 {
			log.Printf("data length not found for matching (got %d)", len(data))
		}
		if data == nil {
			continue
		}
		a.OutputCh <- Diffuse1{Value: ValSeen{
			Start: now,
			ID: []ValID{
				ValID{RFID: data},
			},
		}}
	}
}

func parseRFID(line string) {
}
