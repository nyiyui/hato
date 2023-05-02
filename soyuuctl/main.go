package main

import (
	"github.com/rivo/tview"

	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
)

type LineState struct {
	Power     uint8
	Direction bool
	Brake     bool
}

func (s LineState) String() string {
	var res [5]byte
	if s.Direction {
		res[0] = 'A'
	} else {
		res[0] = 'B'
	}
	if s.Brake {
		res[1] = 'Y'
	} else {
		res[1] = 'N'
	}
	power := strconv.FormatUint(uint64(s.Power), 10)[:3]
	copy(res[2:], power)
	return string(res[:])
}

type Line struct {
	id  rune
	ctl Controller
}

func (l *Line) Commit(s LineState) error {
	var b strings.Builder
	b.WriteString("C")
	b.WriteString(string(l.id))
	b.WriteString(s.String())
	return l.ctl.Commit(b.String())
}

type Controller interface {
	Commit(cmd string) error
}

type SerialController struct {
	f         *os.File
	writeLock sync.Mutex
}

func (c *SerialController) Commit(cmd string) error {
	c.writeLock.Lock()
	defer c.writeLock.Unlock()
	_, err := io.WriteString(c.f, cmd)
	return err
}

type LineEtc struct {
	L    *Line
	Next *LineState
}

type globalState struct {
	Ctl   Controller
	Lines map[string]LineEtc
}

func main2() (err error) {
	var f *os.File
	f, err = os.Open("/dev/ttyACM0")
	if err != nil {
		return fmt.Errorf("open %w", err)
	}
	defer func() {
		err2 := f.Close()
		if err2 != nil {
			err = fmt.Errorf("close %w", err)
		}
	}()
	c := &SerialController{f: f}
	gs := globalState{
		Ctl: c,
		Lines: map[string]LineEtc{
			"A": LineEtc{
				L:    &Line{id: 'A', ctl: c},
				Next: new(LineState),
			},
			"B": LineEtc{
				L:    &Line{id: 'B', ctl: c},
				Next: new(LineState),
			},
		},
	}
	_ = gs

	box := tview.NewBox().SetBorder(true).SetTitle("soyuuctl")
	if err := tview.NewApplication().SetRoot(box, true).Run(); err != nil {
		return err
	}
	return nil
}

func main() {
	err := main2()
	if err != nil {
		log.Fatal(err)
	}
}
