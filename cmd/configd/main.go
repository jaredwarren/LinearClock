package main

import (
	"log"
	"net/http"

	"github.com/jaredwarren/clock/internal/server"
)

const defaultConfigPath = "config.gob"

func main() {
	s := server.New(defaultConfigPath)

	http.HandleFunc("GET /", s.Home)
	http.HandleFunc("GET /events", s.ListEvents)
	http.HandleFunc("POST /events", s.UpdateEvents)
	http.HandleFunc("DELETE /event/{event_id}", s.DeleteEvent)
	http.HandleFunc("GET /test", s.TestHandler)
	http.HandleFunc("POST /config", s.UpdateConfig)
	http.Handle("GET /public/", http.StripPrefix("/public/", http.FileServer(http.Dir("./public"))))

	log.Println("Server listening on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("server: %v", err)
	}
}
