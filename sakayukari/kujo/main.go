package kujo

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/r3labs/sse/v2"
	"nyiyui.ca/hato/sakayukari/tal"
)

type Server struct {
	g *tal.Guide
	s *sse.Server
}

func NewServer(g *tal.Guide) *Server {
	s := &Server{
		g: g,
		s: sse.New(),
	}
	go s.forward()
	//go s.forwardPlatformDisplay()
	return s
}

func (s *Server) forward() {
	s.s.CreateStream("snapshot")
	defer s.s.RemoveStream("snapshot")
	ch := make(chan tal.GuideSnapshot)
	s.g.SnapshotMux.Subscribe("kujo", ch)
	defer s.g.SnapshotMux.Unsubscribe(ch)
	for gs := range ch {
		data, err := json.Marshal(gs)
		if err != nil {
			log.Printf("kujo: marshal json: %s", err)
			continue
		}
		s.s.TryPublish("snapshot", &sse.Event{
			Data: data,
		})
	}
}

func (s *Server) forwardPlatformDisplay() {
	s.s.CreateStream("platform-display")
	defer s.s.RemoveStream("platform-display")
	ch := make(chan tal.GuideSnapshot)
	s.g.SnapshotMux.Subscribe("kujo platform-display", ch)
	defer s.g.SnapshotMux.Unsubscribe(ch)
	for gs := range ch {
		_ = gs
		panic("TODO: get platform-display data and publish")
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.s.ServeHTTP(w, r)
}
