package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/jaredwarren/clock/lib/config"
	"github.com/jaredwarren/clock/lib/display"
)

func (s *Server) TestHandler(w http.ResponseWriter, r *http.Request) {
	c, err := config.ReadConfig("config.gob")
	if err != nil {
		if errorIsMissing(err) {
			c = config.DefaultConfig
		} else {
			fmt.Fprintf(w, "get config error:%+v", err)
			return
		}
	}

	t := time.Now()

	to := r.URL.Query().Get("time-override")
	if to != "" {
		t, err = time.Parse("15:04", to)
		if err != nil {
			fmt.Fprintf(w, "time error:%+v", err)
			return
		}
	}

	hd := display.NewHTMLDisplay(c, w, t)
	err = display.DisplayTime(t, c, hd)
	if err != nil {
		fmt.Fprintf(w, "render error:%+v", err)
		return
	}
}
