package main

import (
	"fmt"
	"net/http"

	"github.com/jaredwarren/clock/lib/server"
)

func main() {
	fmt.Println("Starting...")

	s := server.NewServer()
	http.HandleFunc("GET /", s.Home)
	http.HandleFunc("POST /config", s.UpdateConfig)

	fmt.Println("Server listening on :8080")
	http.ListenAndServe(":8080", nil)

	fmt.Println("Done!")
}
