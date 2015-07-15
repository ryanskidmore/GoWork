package main

import (
	"fmt"
	"github.com/go-martini/martini"
	"github.com/ryanskidmore/GoWork"
	"net/http"
)

func main() {
	ws, _ := gowork.NewServer("w4PYxQjVP9ZStjWpBt5t28CEBmRs8NPx")
	m := martini.Classic()
	m.Get("/register", func() string {
		id, clienttest := ws.Workers.Register(ws)
		return id + "," + clienttest
	})
	m.Post("/verify", func(req *http.Request) string {
		id := req.FormValue("id")
		clientresp := req.FormValue("clientresp")
		auth, err := ws.Workers.Verify(ws, id, clientresp)
		fmt.Println(err)
		return auth
	})
	m.Post("/get_work", func(req *http.Request) string {
		w, err := ws.Get(req.FormValue("id"), req.FormValue("sessionauthkey"))
		fmt.Println(err)
		return w.Marshal()
	})
	m.Run()
}
