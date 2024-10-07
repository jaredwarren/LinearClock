package main

import (
	"fmt"
	"net/http"

	"github.com/jaredwarren/clock/lib/server"
)

func main() {
	fmt.Println("...")

	// TOOD: add lightweight web server. using std-lib
	// https://jvns.ca/blog/2024/09/27/some-go-web-dev-notes/

	s := server.NewServer()

	// Define a handler for the root path ("/")
	http.HandleFunc("GET /", s.Home)
	http.HandleFunc("POST /config", s.UpdateConfig)

	// Start the server listening on port 8080
	fmt.Println("Server listening on :8080")
	http.ListenAndServe(":8080", nil)
}
