package sakuragi

import (
	"embed"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/google/uuid"
	. "nyiyui.ca/hato/sakayukari"
	"nyiyui.ca/hato/sakayukari/tal"
	"nyiyui.ca/hato/sakayukari/tal/layout"
)

//go:embed index.html
var templates embed.FS

type Conf struct {
	Guide  ActorRef
	Model  ActorRef
	Guide2 *tal.Guide
}

type sakuragi struct {
	conf           Conf
	actor          *Actor
	sm             *http.ServeMux
	t              *template.Template
	latestMessage  string
	latestGS       tal.GuideSnapshot
	latestAttitude tal.Attitude
	g              *tal.Guide
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
		g:     conf.Guide2,
	}
	s.t = template.Must(template.New("index").Funcs(sprig.FuncMap()).Funcs(template.FuncMap{
		//"div_int64": func(a, b int64) int64 {
		//	return a / b
		//},
		//"div": func(a, b any) any {
		//	switch a := a.(type) {
		//	case int:
		//		return a / b.(int)
		//	case uint32:
		//		switch b := b.(type) {
		//		case uint32:
		//			return a / b
		//		case int:
		//			return a / uint32(b)
		//		default:
		//			panic("what")
		//		}
		//	case int64:
		//		return a / b.(int64)
		//	default:
		//		panic(fmt.Sprintf("(add %T %T) not supported (yet)", a, b))
		//	}
		//},
		//"add": func(a, b any) any {
		//	switch a := a.(type) {
		//	case int:
		//		return a + b.(int)
		//	case uint32:
		//		return a + b.(uint32)
		//	default:
		//		panic(fmt.Sprintf("(add %T %T) not supported (yet)", a, b))
		//	}
		//},
		//"subtract_int64": func(a, b int64) int64 {
		//	return a - b
		//},
		"map": func(vs ...any) map[string]any {
			if len(vs)%2 != 0 {
				panic("# of args is not even")
			}
			res := map[string]any{}
			for i, v := range vs {
				if i%2 == 0 {
					continue
				}
				name := vs[i-1].(string)
				res[name] = v
			}
			return res
		},
		"oneContains": func(ts []tal.Train, lineI int) bool {
			for _, t := range ts {
				for i := t.CurrentBack; i <= t.CurrentFront; i++ {
					if int(t.Path.Follows[i].LineI) == lineI {
						return true
					}
				}
			}
			return false
		},
		"contains": func(t tal.Train, lineI int) bool {
			for i := t.CurrentBack; i <= t.CurrentFront; i++ {
				if int(t.Path.Follows[i].LineI) == lineI {
					return true
				}
			}
			return false
		},
		"hasValidFormI": func(t tal.Train) bool {
			return t.FormI != (uuid.UUID{})
		},
		"offsetToPos": func(t tal.Train, offset int64) *layout.Position {
			pos, overrun := s.g.Layout.OffsetToPosition(*t.Path, offset)
			if overrun != nil {
				return nil
			}
			return &pos
		},
	}).ParseFS(templates, "*.html"))
	s.setup()
	go s.loop()
	return a
}

func (s *sakuragi) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	err := s.t.ExecuteTemplate(w, "index", map[string]interface{}{
		"msg": s.latestMessage,
		"gs":  s.latestGS,
		"att": s.latestAttitude,
		"now": time.Now().Format("15:04:05"),
		"g":   s.g,
	})
	if err != nil {
		panic(err)
	}
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
		case s.conf.Model:
			switch val := diffuse.Value.(type) {
			case tal.Attitude:
				s.latestAttitude = val
			}
		}
		if msg, ok := diffuse.Value.(Message); ok {
			s.latestMessage = string(msg)
		}
	}
}
