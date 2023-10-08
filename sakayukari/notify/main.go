package notify

import (
	"log"
	"os"
	"runtime/pprof"
	"sync"
	"time"

	"golang.org/x/exp/slices"
)

const multiplexerTimeout = 200 * time.Millisecond

type subscriber[E any] struct {
	ch      chan E
	comment string
}

type MultiplexerSender[E any] struct {
	m *Multiplexer[E]
}

func (ms *MultiplexerSender[E]) Send(e E) {
	go ms.m.send(e)
}

func NewMultiplexerSender[E any](comment string) (*MultiplexerSender[E], *Multiplexer[E]) {
	m := &Multiplexer[E]{
		comment: comment,
	}
	return &MultiplexerSender[E]{m: m}, m
}

type Multiplexer[E any] struct {
	comment         string
	subscribersLock sync.Mutex
	subscribers     []subscriber[E]
}

// subscribersLock must be taken!
func (m *Multiplexer[E]) cleanup() {
	last := len(m.subscribers) - 1
	if m.subscribers[last].ch == nil {
		return
	}
	for i, sub := range m.subscribers {
		if sub.ch == nil {
			m.subscribers[i], m.subscribers[last] = m.subscribers[last], subscriber[E]{}
			return
		}
	}
}

func (m *Multiplexer[E]) Subscribe(comment string, c chan E) {
	m.subscribersLock.Lock()
	defer m.subscribersLock.Unlock()
	sub := subscriber[E]{
		ch:      c,
		comment: comment,
	}
	last := len(m.subscribers) - 1
	if last >= 0 && m.subscribers[last].ch == nil {
		m.subscribers[last] = sub
		m.cleanup()
	} else {
		m.subscribers = append(m.subscribers, sub)
	}
}

func (m *Multiplexer[E]) Unsubscribe(c chan E) {
	m.subscribersLock.Lock()
	defer m.subscribersLock.Unlock()
	i := slices.IndexFunc(m.subscribers, func(sub subscriber[E]) bool { return sub.ch == c })
	if i == -1 {
		panic("already unsubscribed")
	}
	m.subscribers[i] = subscriber[E]{}
	m.cleanup()
}

func (m *Multiplexer[E]) send(e E) {
	m.subscribersLock.Lock()
	defer m.subscribersLock.Unlock()
	for _, sub := range m.subscribers {
		select {
		case sub.ch <- e:
		case <-time.After(multiplexerTimeout):
			m.timeout(sub, e)
		}
	}
}

func (m *Multiplexer[E]) timeout(sub subscriber[E], e E) {
	pprof.Lookup("goroutine").WriteTo(os.Stderr, 1)
	log.Printf("multiplexer %s: subscriber %s timed out: %#v", m.comment, sub.comment, e)
}
