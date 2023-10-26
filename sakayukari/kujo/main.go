package kujo

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"time"

	"github.com/justinas/alice"
	"github.com/r3labs/sse/v2"
	"github.com/rs/cors"
	"nyiyui.ca/hato/sakayukari/notify"
	"nyiyui.ca/hato/sakayukari/tal"
)

type ETAReport struct {
	Station string
	ETA     time.Time
	Op      Operation
}

type Operation struct {
	Type  string
	Index string
	Track string
	Dir   string
}

type Server struct {
	g       *tal.Guide
	s       *sse.Server
	mux     *http.ServeMux
	ETAMuxS *notify.MultiplexerSender[ETAReport]
	etaMux  *notify.Multiplexer[ETAReport]
}

func NewServer(g *tal.Guide) *Server {
	s := &Server{
		g:   g,
		s:   sse.New(),
		mux: http.NewServeMux(),
	}
	s.ETAMuxS, s.etaMux = notify.NewMultiplexerSender[ETAReport]("kujo ETA")
	s.mux.Handle("/sse", s.s)
	s.mux.HandleFunc("/platformdisplay", s.platformDisplay)
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

func (s *Server) platformDisplay(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Type", "text/event-stream")
	var stationID string
	var stationIDSet bool
	if stationIDSet = r.URL.Query().Has("station"); stationIDSet {
		stationID = r.URL.Query().Get("station")
	}
	//if err != nil {
	//	w.WriteHeader(422)
	//	w.WriteString("?station is not a UUID")
	//	return
	//}
	fmt.Fprint(w, "event: status\n")
	fmt.Fprintf(w, "data: ok\n\n")
	w.(http.Flusher).Flush()
	reports := map[string]ETAReport{}
	ch := make(chan ETAReport, 1)
	s.etaMux.Subscribe(fmt.Sprintf("%s from %s", r.URL, r.RemoteAddr), ch)
	defer s.etaMux.Unsubscribe(ch)
	for eta := range ch {
		reports[eta.Op.Index] = eta
		keys := []string{}
		for key := range reports {
			keys = append(keys, key)
		}
		sort.Slice(keys, func(i, j int) bool {
			return reports[keys[i]].ETA.Before(reports[keys[i]].ETA)
		})
		// pick first 2
		allocs := []map[string]interface{}{}
		for _, op := range keys {
			if len(allocs) >= 2 {
				break
			}
			eta := reports[op]
			if stationIDSet && eta.Station != stationID {
				continue
			}
			data := map[string]interface{}{
				"type":  eta.Op.Type,
				"index": eta.Op.Index,
				"time":  eta.ETA.UnixMilli(),
				"track": eta.Op.Track,
				"dir":   eta.Op.Dir,
			}
			allocs = append(allocs, data)
		}
		jsonData, err := json.Marshal(allocs)
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, "marshalling json: %s", err)
			return
		}
		fmt.Fprint(w, "event: updateAlloc\n")
		fmt.Fprintf(w, "data: %s\n\n", jsonData)
		w.(http.Flusher).Flush()
	}
	//ch := make(chan oal.AllStatus)
	//panic("not implemented yet")
	//for as := range ch {
	//	as.Operations
	//	fmt.Fprint(w, "event: updateAlloc\n")
	//	fmt.Fprintf(w, "data: %s\n\n", data)
	//	w.(http.Flusher).Flush()
	//}
}

func (s *Server) Handler() http.Handler {
	return alice.New(cors.Default().Handler).Then(s.mux)
}
