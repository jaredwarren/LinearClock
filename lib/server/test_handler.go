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

	hd := display.NewHTMLDisplay(c, w)

	err = display.DisplayTime(time.Now(), c, hd)
	if err != nil {
		fmt.Fprintf(w, "render error:%+v", err)
		return
	}
}
