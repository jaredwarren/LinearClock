package server

import (
	"net/http"
	"time"

	"github.com/jaredwarren/clock/internal/config"
	"github.com/jaredwarren/clock/internal/display"
)

func (s *Server) TestHandler(w http.ResponseWriter, r *http.Request) {
	c, err := config.ReadConfig(s.ConfigPath)
	if err != nil {
		if errorIsMissing(err) {
			c = config.DefaultConfig.Clone()
		} else {
			http.Error(w, "get config: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	t := time.Now()
	if to := r.URL.Query().Get("time"); to != "" {
		// Full datetime e.g. 2026-03-12T10:15 in local time
		parsed, parseErr := time.ParseInLocation("2006-01-02T15:04", to, time.Local)
		if parseErr != nil {
			http.Error(w, "time: "+parseErr.Error(), http.StatusBadRequest)
			return
		}
		t = parsed
	} else if to := r.URL.Query().Get("time-override"); to != "" {
		// HH:MM = today at that time in local
		parsed, parseErr := time.ParseInLocation("15:04", to, time.Local)
		if parseErr != nil {
			http.Error(w, "time-override: "+parseErr.Error(), http.StatusBadRequest)
			return
		}
		now := time.Now()
		t = time.Date(now.Year(), now.Month(), now.Day(), parsed.Hour(), parsed.Minute(), 0, 0, time.Local)
	}

	// Avoid serving a cached page so refresh during an event window shows current colors
	w.Header().Set("Cache-Control", "no-store")

	hd := display.NewHTMLDisplay(c, w, t)
	if err := display.DisplayTime(t, c, hd); err != nil {
		http.Error(w, "render: "+err.Error(), http.StatusInternalServerError)
		return
	}
}
