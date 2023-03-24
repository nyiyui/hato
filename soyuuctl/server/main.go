package server

/*
import (
	"net/http"

	"github.com/rs/cors"
	"nyiyui.ca/soyuu/soyuuctl/conn"
)

var stNames = []string{
	"0",
}

type Server struct {
}

func (s *Server) stHook(name string, v conn.Val) {
	switch v := v.(type) {
	case conn.ValAttitude:
		switch v.State {
		case conn.STStateSide:
			s.LineReq("B", conn.ReqLine{
				Brake:     false,
				Direction: true,
				Power:     00,
			})
		case conn.STStateTop:
			s.LineReq("B", conn.ReqLine{
				Brake:     false,
				Direction: true,
				Power:     50,
			})
		case conn.STStateBase:
			s.LineReq("B", conn.ReqLine{
				Brake:     false,
				Direction: true,
				Power:     80,
			})
		}
	}
}

func RunServer(cs *conn.State) {
	sts := map[string]*conn.Conn{}
	for _, stName := range stNames {
		var c *conn.Conn
		ok := false
		for !ok {
			c, ok = cs.GetST(stName)
		}
		func() {
			c.HooksLock.Lock()
			defer c.HooksLock.Unlock()
			c.Hooks = append(c.Hooks, func(v conn.Val) {
				s.stHook(stName, v)
			})
		}()
		sts[stName] = c
	}
	corsHandler := cors.New(cors.Options{
		AllowedOrigins: []string{"http://foo.com"},
	})
	mux := http.NewServeMux()
	mux.HandleFunc("/sts", func(w http.ResponseWriter, r *http.Request) {
		for name, c := range sts {
		}
	})
	&http.Server{
		Addr:    ":8032",
		Handler: corsHandler.Handler(mux),
	}
}
*/
