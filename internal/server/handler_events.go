package server

import (
	"crypto/rand"
	"encoding/hex"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/jaredwarren/clock/internal/config"
)

const dateTimeLayout = "2006-01-02T15:04"

// eventListData is passed to the events template (list page).
type eventListData struct {
	NavActive string
	Events    []config.TickEvent
	Config    *config.Config // for nav; can be nil
	LastIndex int             // len(Events)-1 for move-down disable
}

func (s *Server) ListEvents(w http.ResponseWriter, r *http.Request) {
	c, err := config.ReadConfig(s.ConfigPath)
	if err != nil {
		if errorIsMissing(err) {
			c = config.DefaultConfig.Clone()
		} else {
			http.Error(w, "get config: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
	// Ensure Events slice is non-nil for template range.
	events := c.Tick.Events
	if events == nil {
		events = []config.TickEvent{}
	}

	lastIndex := 0
	if n := len(events); n > 0 {
		lastIndex = n - 1
	}
	data := eventListData{NavActive: "events", Events: events, Config: c, LastIndex: lastIndex}
	files := []string{
		"templates/events.html",
		"templates/layout.html",
	}
	tmpl, err := template.New("base").Funcs(template.FuncMap{
		"ColorString":     ColorString,
		"TimeFormat":      TimeFormat,
		"TimeFormatLocal": TimeFormatLocal,
	}).ParseFiles(files...)
	if err != nil {
		http.Error(w, "parse template: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "exec template: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) UpdateEvents(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "parse form: "+err.Error(), http.StatusBadRequest)
		return
	}

	c, err := config.ReadConfig(s.ConfigPath)
	if err != nil {
		if errorIsMissing(err) {
			c = config.DefaultConfig.Clone()
		} else {
			http.Error(w, "get config: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if c.Tick.Events == nil {
		c.Tick.Events = []config.TickEvent{}
	}

	// datetime-local sends local time with no zone; parse as local so daily events
	// compare time-of-day in the same zone as time.Now().
	startStr := r.FormValue("event.start")
	start, err := time.ParseInLocation(dateTimeLayout, startStr, time.Local)
	if err != nil {
		http.Error(w, "invalid event.start: "+err.Error(), http.StatusBadRequest)
		return
	}
	endStr := r.FormValue("event.end")
	end, err := time.ParseInLocation(dateTimeLayout, endStr, time.Local)
	if err != nil {
		http.Error(w, "invalid event.end: "+err.Error(), http.StatusBadRequest)
		return
	}
	if end.Before(start) {
		http.Error(w, "event.end must be after event.start", http.StatusBadRequest)
		return
	}

	repeat := r.FormValue("event.repeat")
	if repeat != config.RepeatNone && repeat != config.RepeatDaily {
		repeat = config.RepeatNone
	}

	evt := config.TickEvent{
		ID:     generateEventID(),
		Title:  r.FormValue("event.title"),
		Start:  start,
		End:    end,
		Repeat: repeat,
	}
	if v := r.FormValue("event.past-color"); v != "" {
		color, err := hexStringToUint32(v)
		if err != nil {
			http.Error(w, "event.past-color: "+err.Error(), http.StatusBadRequest)
			return
		}
		evt.PastColorOverride = color
	}
	if v := r.FormValue("event.present-color"); v != "" {
		color, err := hexStringToUint32(v)
		if err != nil {
			http.Error(w, "event.present-color: "+err.Error(), http.StatusBadRequest)
			return
		}
		evt.PresentColorOverride = color
	}
	if v := r.FormValue("event.future-color"); v != "" {
		color, err := hexStringToUint32(v)
		if err != nil {
			http.Error(w, "event.future-color: "+err.Error(), http.StatusBadRequest)
			return
		}
		evt.FutureColorOverride = color
	}
	if v := r.FormValue("event.future-color-b"); v != "" {
		color, err := hexStringToUint32(v)
		if err != nil {
			http.Error(w, "event.future-color-b: "+err.Error(), http.StatusBadRequest)
			return
		}
		evt.FutureColorBOverride = color
	}

	c.Tick.Events = append(c.Tick.Events, evt)
	if err := c.Validate(); err != nil {
		http.Error(w, "invalid config: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err := WriteConfigLocked(s.ConfigPath, c); err != nil {
		http.Error(w, "write config: "+err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/events", http.StatusSeeOther)
}

func (s *Server) DeleteEvent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("event_id")
	if id == "" {
		http.Error(w, "missing event_id", http.StatusBadRequest)
		return
	}
	s.deleteEventByID(w, r, id)
}

// DeleteEventPost handles POST /events/delete with form field event_id (for form-based delete).
func (s *Server) DeleteEventPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "parse form: "+err.Error(), http.StatusBadRequest)
		return
	}
	id := r.FormValue("event_id")
	if id == "" {
		http.Redirect(w, r, "/events", http.StatusSeeOther)
		return
	}
	s.deleteEventByID(w, r, id)
}

// EditEvent handles POST /events/edit to update an existing event by ID.
func (s *Server) EditEvent(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "parse form: "+err.Error(), http.StatusBadRequest)
		return
	}
	id := r.FormValue("event_id")
	if id == "" {
		http.Error(w, "missing event_id", http.StatusBadRequest)
		return
	}

	c, err := config.ReadConfig(s.ConfigPath)
	if err != nil {
		if errorIsMissing(err) {
			http.Redirect(w, r, "/events", http.StatusSeeOther)
			return
		}
		http.Error(w, "get config: "+err.Error(), http.StatusInternalServerError)
		return
	}
	events := c.Tick.Events
	if len(events) == 0 {
		http.Redirect(w, r, "/events", http.StatusSeeOther)
		return
	}

	var idx = -1
	for i, e := range events {
		if e.ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		http.Redirect(w, r, "/events", http.StatusSeeOther)
		return
	}

	// datetime-local sends local time with no zone; parse as local so daily events
	// compare time-of-day in the same zone as time.Now().
	startStr := r.FormValue("event.start")
	start, err := time.ParseInLocation(dateTimeLayout, startStr, time.Local)
	if err != nil {
		http.Error(w, "invalid event.start: "+err.Error(), http.StatusBadRequest)
		return
	}
	endStr := r.FormValue("event.end")
	end, err := time.ParseInLocation(dateTimeLayout, endStr, time.Local)
	if err != nil {
		http.Error(w, "invalid event.end: "+err.Error(), http.StatusBadRequest)
		return
	}
	if end.Before(start) {
		http.Error(w, "event.end must be after event.start", http.StatusBadRequest)
		return
	}

	repeat := r.FormValue("event.repeat")
	if repeat != config.RepeatNone && repeat != config.RepeatDaily {
		repeat = config.RepeatNone
	}

	evt := &events[idx]
	evt.Title = r.FormValue("event.title")
	evt.Start = start
	evt.End = end
	evt.Repeat = repeat

	if v := r.FormValue("event.past-color"); v != "" {
		color, err := hexStringToUint32(v)
		if err != nil {
			http.Error(w, "event.past-color: "+err.Error(), http.StatusBadRequest)
			return
		}
		evt.PastColorOverride = color
	}
	if v := r.FormValue("event.present-color"); v != "" {
		color, err := hexStringToUint32(v)
		if err != nil {
			http.Error(w, "event.present-color: "+err.Error(), http.StatusBadRequest)
			return
		}
		evt.PresentColorOverride = color
	}
	if v := r.FormValue("event.future-color"); v != "" {
		color, err := hexStringToUint32(v)
		if err != nil {
			http.Error(w, "event.future-color: "+err.Error(), http.StatusBadRequest)
			return
		}
		evt.FutureColorOverride = color
	}
	if v := r.FormValue("event.future-color-b"); v != "" {
		color, err := hexStringToUint32(v)
		if err != nil {
			http.Error(w, "event.future-color-b: "+err.Error(), http.StatusBadRequest)
			return
		}
		evt.FutureColorBOverride = color
	}

	c.Tick.Events = events
	if err := c.Validate(); err != nil {
		http.Error(w, "invalid config: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err := WriteConfigLocked(s.ConfigPath, c); err != nil {
		http.Error(w, "write config: "+err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/events", http.StatusSeeOther)
}

func (s *Server) deleteEventByID(w http.ResponseWriter, r *http.Request, id string) {
	c, err := config.ReadConfig(s.ConfigPath)
	if err != nil {
		if errorIsMissing(err) {
			http.Redirect(w, r, "/events", http.StatusSeeOther)
			return
		}
		http.Error(w, "get config: "+err.Error(), http.StatusInternalServerError)
		return
	}
	newEvents := make([]config.TickEvent, 0, len(c.Tick.Events))
	for _, e := range c.Tick.Events {
		if e.ID != id {
			newEvents = append(newEvents, e)
		}
	}
	c.Tick.Events = newEvents
	if err := c.Validate(); err != nil {
		http.Error(w, "invalid config: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err := WriteConfigLocked(s.ConfigPath, c); err != nil {
		http.Error(w, "write config: "+err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/events", http.StatusSeeOther)
}

func (s *Server) MoveEventUp(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "parse form: "+err.Error(), http.StatusBadRequest)
		return
	}
	idx, err := strconv.Atoi(r.FormValue("index"))
	if err != nil || idx <= 0 {
		http.Error(w, "invalid index", http.StatusBadRequest)
		return
	}
	moveEventByIndex(w, r, s, idx, idx-1)
}

func (s *Server) MoveEventDown(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "parse form: "+err.Error(), http.StatusBadRequest)
		return
	}
	idx, err := strconv.Atoi(r.FormValue("index"))
	if err != nil || idx < 0 {
		http.Error(w, "invalid index", http.StatusBadRequest)
		return
	}
	moveEventByIndex(w, r, s, idx, idx+1)
}

func moveEventByIndex(w http.ResponseWriter, r *http.Request, s *Server, from, to int) {
	c, err := config.ReadConfig(s.ConfigPath)
	if err != nil {
		if errorIsMissing(err) {
			http.Redirect(w, r, "/events", http.StatusSeeOther)
			return
		}
		http.Error(w, "get config: "+err.Error(), http.StatusInternalServerError)
		return
	}
	events := c.Tick.Events
	if events == nil || from < 0 || to < 0 || from >= len(events) || to >= len(events) || from == to {
		http.Redirect(w, r, "/events", http.StatusSeeOther)
		return
	}
	events[from], events[to] = events[to], events[from]
	c.Tick.Events = events
	if err := c.Validate(); err != nil {
		http.Error(w, "invalid config: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err := WriteConfigLocked(s.ConfigPath, c); err != nil {
		http.Error(w, "write config: "+err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/events", http.StatusSeeOther)
}

func generateEventID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "evt_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	return "evt_" + hex.EncodeToString(b)
}
