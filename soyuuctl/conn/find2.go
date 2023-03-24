package conn

import (
	"bufio"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
)

// connect2 connects to a serial port on specified path and creates a new Conn for it.
// State.connsLock must be locked at call site.
func (s *State) connect2(path string) {
	log.Printf("connecting to %s", path)
	cmd := exec.Command("./serial-proxy", path)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Printf("connect %s: StdinPipe: %s", path, err)
		return
	}
	defer stdin.Close()
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("connect %s: StdoutPipe: %s", path, err)
		return
	}
	defer stdout.Close()
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		log.Printf("connect %s: %s", path, err)
		return
	}
	defer cmd.Process.Signal(os.Interrupt)
	_, err = stdin.Write([]byte("I\r\n"))
	if err != nil {
		log.Printf("connect %s: %s", path, err)
		return
	}
	reader := bufio.NewReader(stdout)
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
	s.handleConn(path, &combinedReadWriter{
		r: stdout,
		w: stdin,
	}, c)
}

type combinedReadWriter struct {
	r io.Reader
	w io.Writer
}

func (c *combinedReadWriter) Read(p []byte) (n int, err error)  { return c.r.Read(p) }
func (c *combinedReadWriter) Write(p []byte) (n int, err error) { return c.w.Write(p) }
