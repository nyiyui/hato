package conn

import (
	"bufio"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

func (s *State) find() error {
	matches, err := filepath.Glob("/dev/ttyACM*")
	if err != nil {
		return err
	}
	s.connsLock.Lock()
	defer s.connsLock.Unlock()
	var wg sync.WaitGroup
	for _, match := range matches {
		if _, ok := s.conns[match]; ok {
			continue
		}
		wg.Add(1)
		go s.connect(&wg, match)
	}
	wg.Wait()
	return nil
}

// connect connects to a serial port on specified path and creates a new Conn for it.
// State.connsLock must be locked at call site.
func (s *State) connect(wg *sync.WaitGroup, path string) {
	doneSent := false
	defer func() {
		if !doneSent {
			wg.Done()
		}
	}()
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
	time.Sleep(1 * time.Second)
	_, err = stdin.Write([]byte("I\n"))
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
	id := ParseId(line)
	// log.Printf("connected to %s %s", path, id)
	c := &Conn{
		Id:   id,
		Path: path,
		F:    &combinedReadWriter{r: stdout, w: stdin},
	}
	// TODO: flush stdin on write
	s.conns[path] = c
	wg.Done()
	doneSent = true
	s.handleConn(c)
}

type combinedReadWriter struct {
	r io.Reader
	w io.Writer
}

func (c *combinedReadWriter) Read(p []byte) (n int, err error)  { return c.r.Read(p) }
func (c *combinedReadWriter) Write(p []byte) (n int, err error) { return c.w.Write(p) }
