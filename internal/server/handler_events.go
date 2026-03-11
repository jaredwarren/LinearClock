package server

import (
	"fmt"
	"html/template"
	"net/http"
	"time"
)

// Event represents a calendar event for the clock.
type Event struct {
	Time  time.Time
	Color uint32
}

// Events is the in-memory event list (for demo; replace with persistence later).
var Events = []*Event{}

func (s *Server) ListEvents(w http.ResponseWriter, r *http.Request) {
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
