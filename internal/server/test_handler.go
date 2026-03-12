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
	if to := r.URL.Query().Get("time-override"); to != "" {
		var parseErr error
		t, parseErr = time.Parse("15:04", to)
		if parseErr != nil {
			http.Error(w, "time-override: "+parseErr.Error(), http.StatusBadRequest)
			return
		}
	}

	hd := display.NewHTMLDisplay(c, w, t)
	if err := display.DisplayTime(t, c, hd); err != nil {
		http.Error(w, "render: "+err.Error(), http.StatusInternalServerError)
		return
	}
}
