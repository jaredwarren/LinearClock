package server

import (
	"html/template"
	"net/http"
	"sync"
	"time"
)

// Event represents a calendar event for the clock.
type Event struct {
	Time  time.Time
	Color uint32
}

// Events is the in-memory event list (for demo; replace with persistence later).
var (
	events   = []*Event{}
	eventsMu sync.RWMutex
)

func (s *Server) ListEvents(w http.ResponseWriter, r *http.Request) {
	eventsMu.Lock()
	if len(events) == 0 {
		events = append(events, &Event{
			Time:  time.Now(),
			Color: 0xFF0000,
		})
	}
	snapshot := append([]*Event(nil), events...)
	eventsMu.Unlock()

	files := []string{
		"templates/events.html",
		"templates/layout.html",
	}
	tmpl, err := template.New("base").Funcs(template.FuncMap{
		"ColorString": ColorString,
		"TimeFormat":  TimeFormat,
	}).ParseFiles(files...)
	if err != nil {
		http.Error(w, "parse template: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, snapshot); err != nil {
		http.Error(w, "exec template: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) UpdateEvents(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "parse form: "+err.Error(), http.StatusBadRequest)
		return
	}

	e := &Event{}

	dateTimeStr := r.FormValue("event.time")
	dateTime, err := time.Parse("2006-01-02T15:04", dateTimeStr)
	if err != nil {
		http.Error(w, "invalid event.time: "+err.Error(), http.StatusBadRequest)
		return
	}
	e.Time = dateTime

	v := r.FormValue("event.color")
	color, err := hexStringToUint32(v)
	if err != nil {
		http.Error(w, "event.color: "+err.Error(), http.StatusBadRequest)
		return
	}
	e.Color = color

	eventsMu.Lock()
	events = append(events, e)
	eventsMu.Unlock()

	http.Redirect(w, r, "/events", http.StatusSeeOther)
}

func (s *Server) DeleteEvent(w http.ResponseWriter, r *http.Request) {
	eventsMu.Lock()
	defer eventsMu.Unlock()
	// TODO: parse event_id from path and remove from events
}
