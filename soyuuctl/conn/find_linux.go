package conn

import (
	"bufio"
	"log"
	"path/filepath"
	"strings"

	"github.com/albenik/go-serial/v2"
)

func (s *State) find() error {
	matches, err := filepath.Glob("/dev/ttyACM*")
	if err != nil {
		return err
	}
	s.connsLock.Lock()
	defer s.connsLock.Unlock()
	for _, match := range matches {
		if _, ok := s.conns[match]; ok {
			continue
		}
		// go s.connect(match)
		go s.connect2(match)
	}
	return nil
}

// connect connects to a serial port on specified path and creates a new Conn for it.
// State.connsLock must be locked at call site.
func (s *State) connect(path string) {
	log.Printf("connecting to %s", path)
	port, err := serial.Open(path,
		serial.WithReadTimeout(1000),
	)
	if err != nil {
		log.Printf("connect %s: %s", path, err)
		return
	}
	defer port.Close() // ignore error
	// err = port.SetDTR(true)
	// if err != nil {
	// 	log.Printf("connect %s: set dtr: %s", path, err)
	// 	return
	// }
	_, err = port.Write([]byte("I\r\n"))
	if err != nil {
		log.Printf("connect %s: %s", path, err)
		return
	}
	reader := bufio.NewReader(port)
	var line string
	for !strings.HasPrefix(line, " I") {
		// log.Printf("connect %s: read %s", path, strconv.Quote(line))
		line, err = reader.ReadString('\n')
		if err != nil {
			log.Printf("connect %s: reading id: %s", path, err)
			return
		}
	}
	if len(line) <= 2 {
		log.Printf("connect %s: not enough id: %s", path, line)
		return
	}
	line = strings.TrimSpace(line[2:])
	id := parseId(line)
	log.Printf("connected to %s %s", path, id)
	reqs := make(chan Req)
	c := &Conn{
		Id:   id,
		Reqs: reqs,
	}
	s.conns[path] = c
	s.handleConn(path, port, c)
}
