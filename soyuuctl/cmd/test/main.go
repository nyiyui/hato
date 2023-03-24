package main

import (
	"log"

	"nyiyui.ca/soyuu/soyuuctl/conn"
)

var s *conn.State

func stHook(name string, v conn.Val) {
	log.Printf("stHook %#v", v)
	defer log.Printf("stHookEnd") // happend immediately after
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
				Power:     80,
			})
		case conn.STStateBase:
			s.LineReq("B", conn.ReqLine{
				Brake:     false,
				Direction: true,
				Power:     120,
			})
		}
	}
}

func main2() error {
	s = conn.NewState()
	err := s.Find()
	if err != nil {
		log.Fatalf("find: %s", err)
	}
	var c *conn.Conn
	ok := false
	for !ok {
		c, ok = s.GetST("0")
	}
	func() {
		c.HooksLock.Lock()
		defer c.HooksLock.Unlock()
		c.Hooks = append(c.Hooks, func(v conn.Val) { stHook("0", v) })
	}()
	s.LineReq("A", conn.ReqLine{
		Brake:     false,
		Direction: true,
		Power:     00,
	})
	s.LineReq("B", conn.ReqLine{
		Brake:     false,
		Direction: true,
		Power:     00,
	})
	select {}
	/*
		for {
			v, err := s.STVal("0")
			if err != nil {
				log.Print(err)
				continue
			}
			log.Printf("%#v", v)
			switch v := v.(type) {
			case conn.ValAttitude:
				switch v.State {
				case conn.STStateSide:
					s.LineReq("B", conn.ReqLine{
						Brake:     false,
						Direction: true,
						Power:     00,
					})
					time.Sleep(1 * time.Second)
					s.LineReq("B", conn.ReqLine{
						Brake:     false,
						Direction: true,
						Power:     50,
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
	*/
	return nil
}

func main() {
	err := main2()
	if err != nil {
		log.Fatal(err)
	}
}
