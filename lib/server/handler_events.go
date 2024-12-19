package server

import (
	"fmt"
	"html/template"
	"net/http"
	"time"
)

type Event struct {
	Time  time.Time
	Color uint32
}

var Events = []*Event{}

func (s *Server) ListEvents(w http.ResponseWriter, r *http.Request) {
	// c, err := config.ReadConfig("config.gob")
	// if err != nil {
	// 	if errorIsMissing(err) {
	// 		c = config.DefaultConfig
	// 	} else {
	// 		fmt.Fprintf(w, "get config error:%+v", err)
	// 		return
	// 	}
	// }

	// for testing
	if len(Events) == 0 {
		Events = append(Events, &Event{
			Time:  time.Now(),
			Color: 0xFF0000,
		})
	}

	files := []string{
		"templates/events.html",
		"templates/layout.html",
	}
	tmpl, err := template.New("base").Funcs(template.FuncMap{
		"ColorString": ColorString,
		"TimeFormat":  TimeFormat,
	}).ParseFiles(files...)
	if err != nil {
		fmt.Fprintf(w, "parse template error:%+v", err)
		return
	}
	err = tmpl.Execute(w, Events)
	if err != nil {
		fmt.Fprintf(w, "exec temp error:%+v", err)
		return
	}
}

func (s *Server) UpdateEvents(w http.ResponseWriter, r *http.Request) {
	// Parse the form data
	err := r.ParseForm()
	if err != nil {
		fmt.Fprintf(w, "Error parsing form data: %v", err)
		return
	}

	fmt.Printf("~~~~~~~~~~~~~~~\n %+v\n\n", r.Form)

	e := &Event{}

	{
		dateTimeStr := r.FormValue("event.time")
		fmt.Println("Received datetime:", dateTimeStr)
		// Parse the datetime string into a time.Time object
		dateTime, err := time.Parse("2006-01-02T15:04", dateTimeStr)
		if err != nil {
			http.Error(w, "Error parsing datetime", http.StatusBadRequest)
			return
		}

		fmt.Println("Parsed datetime:", dateTime)
		e.Time = dateTime
	}

	fmt.Println("Num.PresentColor:" + r.FormValue("event.color"))
	{
		v := r.FormValue("event.color")
		color, err := hexStringToUint32(v)
		if err != nil {
			fmt.Fprintf(w, "convert 'num.present-color' error (%s):%+v", v, err)
			return
		}
		e.Color = color
	}

	Events = append(Events, e)

	http.Redirect(w, r, "/events", http.StatusSeeOther)
}

func (s *Server) DeleteEvent(w http.ResponseWriter, r *http.Request) {}
