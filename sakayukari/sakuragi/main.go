package sakuragi

import (
	"embed"
	"html/template"
	"log"
	"net/http"

	. "nyiyui.ca/hato/sakayukari"
	"nyiyui.ca/hato/sakayukari/tal"
)

//go:embed index.html
var templates embed.FS

type Conf struct {
	Guide ActorRef
	Model ActorRef
}

type sakuragi struct {
	conf     Conf
	actor    *Actor
	sm       *http.ServeMux
	t        *template.Template
	latestGS tal.GuideSnapshot
}

func Sakuragi(conf Conf) *Actor {
	a := &Actor{
		Comment:  "sakuragi",
		InputCh:  make(chan Diffuse1),
		OutputCh: make(chan Diffuse1),
		Inputs:   []ActorRef{conf.Guide, conf.Model},
		Type: ActorType{
			Input:       true,
			LinearInput: true,
			Output:      true,
		},
	}
	s := &sakuragi{
		conf:  conf,
		actor: a,
		sm:    http.NewServeMux(),
		t:     template.Must(template.New("index").ParseFS(templates, "*.html")),
	}
	s.setup()
	go s.loop()
	return a
}

func (s *sakuragi) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	s.t.ExecuteTemplate(w, "index", map[string]interface{}{
		"gs": s.latestGS,
	})
}

func (s *sakuragi) setup() {
	s.sm.HandleFunc("/", s.handleIndex)
	go func() {
		err := http.ListenAndServe("0.0.0.0:8080", s.sm)
		log.Fatalf("sakuragi: %s", err)
	}()
}

func (s *sakuragi) loop() {
	for diffuse := range s.actor.InputCh {
		switch diffuse.Origin {
		case s.conf.Guide:
			switch val := diffuse.Value.(type) {
			case tal.GuideSnapshot:
				s.latestGS = val
			}
		}
	}
}
