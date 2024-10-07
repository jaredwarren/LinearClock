package server

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/template"

	"github.com/davecgh/go-spew/spew"
	"github.com/jaredwarren/clock/lib/config"
)

type Server struct {
}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) Home(w http.ResponseWriter, r *http.Request) {
	c, err := config.ReadConfig("config.gob")
	if err != nil {
		fmt.Fprintf(w, "get config error:%+v", err)
		return
	}

	fmt.Println(os.Getwd())
	tmpl, err := template.ParseFiles("templates/home.html")
	if err != nil {
		panic(err)
	}
	err = tmpl.Execute(w, c)
	if err != nil {
		panic(err)
	}
}

func (s *Server) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	// Parse the form data
	err := r.ParseForm()
	if err != nil {
		fmt.Fprintf(w, "Error parsing form data: %v", err)
		return
	}

	spew.Dump(r.Form)

	// Access form values
	name := r.FormValue("brightness")
	email := r.FormValue("color")

	// Do something with the form data, e.g., print it to the console
	fmt.Println("Name:", name)
	fmt.Println("Email:", email)

	// TODO: validate
	c, err := config.ReadConfig("config.gob")
	if err != nil {
		fmt.Fprintf(w, "get config error:%+v", err)
		return
	}

	tpci := r.FormValue("tick.past-color")
	tpc, err := hexStringToUint32(tpci)
	if err != nil {
		fmt.Fprintf(w, "convert 'tick.past-color' error (%s):%+v", tpci, err)
		return
	}
	c.Tick.PastColor = tpc

	// TODO: write config
}

func hexStringToUint32(hexStr string) (uint32, error) {
	hexStr = strings.TrimPrefix(hexStr, "#")

	r, err := strconv.ParseUint(hexStr[0:2], 16, 8)
	if err != nil {
		return 0, fmt.Errorf("parse color x in r - %w", err)
	}
	g, err := strconv.ParseUint(hexStr[2:4], 16, 8)
	if err != nil {
		return 0, fmt.Errorf("parse color x in g - %w", err)
	}
	b, err := strconv.ParseUint(hexStr[4:6], 16, 8)
	if err != nil {
		return 0, fmt.Errorf("parse color x in b - %w", err)
	}

	return uint32(r)<<16 | uint32(g)<<8 | uint32(b), nil
}
