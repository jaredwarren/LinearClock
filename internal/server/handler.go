package server

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jaredwarren/clock/internal/config"
)

// Server is the config HTTP server. ConfigPath is the path to the shared config file.
type Server struct {
	ConfigPath string
}

// New returns a Server that reads/writes config at configPath.
func New(configPath string) *Server {
	return &Server{ConfigPath: configPath}
}

// homePageData is passed to the home template: NavActive drives the shared nav; *config.Config is embedded
// so existing templates can keep using {{.Brightness}}, {{.Tick}}, etc.
type homePageData struct {
	NavActive string
	*config.Config
}

func (s *Server) Home(w http.ResponseWriter, r *http.Request) {
	c, err := config.ReadConfig(s.ConfigPath)
	if err != nil {
		if errorIsMissing(err) {
			c = config.DefaultConfig.Clone()
		} else {
			http.Error(w, "get config: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	files := []string{
		"templates/home.html",
		"templates/layout.html",
	}
	tmpl, err := template.New("base").Funcs(template.FuncMap{
		"ColorString": ColorString,
		"TimeNum":     TimeNum,
	}).ParseFiles(files...)
	if err != nil {
		http.Error(w, "parse template: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, homePageData{NavActive: "config", Config: c}); err != nil {
		http.Error(w, "exec template: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func ColorString(color uint32) string {
	return fmt.Sprintf("#%06X", color)
}

func TimeNum(t time.Duration) string {
	return fmt.Sprintf("%d", int(t.Seconds()))
}

func TimeFormat(t time.Time) string {
	return t.Format("2006-01-02T15:04")
}

func errorIsMissing(err error) bool {
	return strings.Contains(err.Error(), "no such file or directory")
}

func (s *Server) UpdateConfig(w http.ResponseWriter, r *http.Request) {
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

	// Brightness: 0-256 (display uses 0-256 for scale)
	if i, err := strconv.Atoi(r.FormValue("brightness")); err != nil {
		http.Error(w, "invalid brightness: "+err.Error(), http.StatusBadRequest)
		return
	} else if i < 0 || i > 256 {
		http.Error(w, "brightness must be 0-256", http.StatusBadRequest)
		return
	} else {
		c.Brightness = i
	}

	// Refresh rate: 1-900 seconds
	if i, err := strconv.ParseInt(r.FormValue("refresh-rate"), 10, 64); err != nil {
		http.Error(w, "invalid refresh-rate: "+err.Error(), http.StatusBadRequest)
		return
	} else if i < 1 || i > 900 {
		http.Error(w, "refresh-rate must be 1-900 seconds", http.StatusBadRequest)
		return
	} else {
		c.RefreshRate = time.Second * time.Duration(i)
	}

	// Gap: non-negative, cap at 100
	if i, err := strconv.Atoi(r.FormValue("gap")); err != nil {
		http.Error(w, "invalid gap: "+err.Error(), http.StatusBadRequest)
		return
	} else if i < 0 || i > 100 {
		http.Error(w, "gap must be 0-100", http.StatusBadRequest)
		return
	} else {
		c.Gap = i
	}

	// Calendar/iCal settings
	if _, ok := r.Form["calendar.ical-url"]; ok {
		// URL may be intentionally cleared to disable iCal sync.
		c.Calendar.ICalURL = strings.TrimSpace(r.FormValue("calendar.ical-url"))
	}

	if v, ok := r.Form["calendar.poll-interval-seconds"]; ok && len(v) > 0 && strings.TrimSpace(v[0]) != "" {
		if i, err := strconv.Atoi(v[0]); err != nil {
			http.Error(w, "invalid calendar.poll-interval-seconds: "+err.Error(), http.StatusBadRequest)
			return
		} else if i < 10 || i > 86400 {
			http.Error(w, "calendar.poll-interval-seconds must be 10-86400", http.StatusBadRequest)
			return
		} else {
			c.Calendar.PollIntervalSeconds = i
		}
	}

	if v, ok := r.Form["calendar.lookback-days"]; ok && len(v) > 0 && strings.TrimSpace(v[0]) != "" {
		if i, err := strconv.Atoi(v[0]); err != nil {
			http.Error(w, "invalid calendar.lookback-days: "+err.Error(), http.StatusBadRequest)
			return
		} else if i < 0 || i > 365 {
			http.Error(w, "calendar.lookback-days must be 0-365", http.StatusBadRequest)
			return
		} else {
			c.Calendar.LookbackDays = i
		}
	}

	if v, ok := r.Form["calendar.lookahead-days"]; ok && len(v) > 0 && strings.TrimSpace(v[0]) != "" {
		if i, err := strconv.Atoi(v[0]); err != nil {
			http.Error(w, "invalid calendar.lookahead-days: "+err.Error(), http.StatusBadRequest)
			return
		} else if i < 0 || i > 365 {
			http.Error(w, "calendar.lookahead-days must be 0-365", http.StatusBadRequest)
			return
		} else {
			c.Calendar.LookaheadDays = i
		}
	}

	// LED overrides for generated iCal events. Input comes as "#RRGGBB";
	// hexStringToUint32 returns 0 for black, which means "no override".
	if v, ok := r.Form["calendar.override-past-color"]; ok && len(v) > 0 && strings.TrimSpace(v[0]) != "" {
		color, err := hexStringToUint32(v[0])
		if err != nil {
			http.Error(w, "calendar.override-past-color: "+err.Error(), http.StatusBadRequest)
			return
		}
		c.Calendar.OverridePastColor = color
	}
	if v, ok := r.Form["calendar.override-present-color"]; ok && len(v) > 0 && strings.TrimSpace(v[0]) != "" {
		color, err := hexStringToUint32(v[0])
		if err != nil {
			http.Error(w, "calendar.override-present-color: "+err.Error(), http.StatusBadRequest)
			return
		}
		c.Calendar.OverridePresentColor = color
	}
	if v, ok := r.Form["calendar.override-future-color"]; ok && len(v) > 0 && strings.TrimSpace(v[0]) != "" {
		color, err := hexStringToUint32(v[0])
		if err != nil {
			http.Error(w, "calendar.override-future-color: "+err.Error(), http.StatusBadRequest)
			return
		}
		c.Calendar.OverrideFutureColor = color
	}
	if v, ok := r.Form["calendar.override-future-b-color"]; ok && len(v) > 0 && strings.TrimSpace(v[0]) != "" {
		color, err := hexStringToUint32(v[0])
		if err != nil {
			http.Error(w, "calendar.override-future-b-color: "+err.Error(), http.StatusBadRequest)
			return
		}
		c.Calendar.OverrideFutureBColor = color
	}

	// Tick.StartLed: non-negative
	if i, err := strconv.Atoi(r.FormValue("tick.start-led")); err != nil {
		http.Error(w, "invalid tick.start-led: "+err.Error(), http.StatusBadRequest)
		return
	} else if i < 0 {
		http.Error(w, "tick.start-led must be non-negative", http.StatusBadRequest)
		return
	} else {
		c.Tick.StartLed = i
	}

	// Tick.TicksPerHour: 1-60
	if i, err := strconv.Atoi(r.FormValue("tick.ticks-per-hour")); err != nil {
		http.Error(w, "invalid tick.ticks-per-hour: "+err.Error(), http.StatusBadRequest)
		return
	} else if i < 1 || i > 60 {
		http.Error(w, "tick.ticks-per-hour must be 1-60", http.StatusBadRequest)
		return
	} else {
		c.Tick.TicksPerHour = i
	}

	// Tick.NumHours: 1-24
	if i, err := strconv.Atoi(r.FormValue("tick.num-hours")); err != nil {
		http.Error(w, "invalid tick.num-hours: "+err.Error(), http.StatusBadRequest)
		return
	} else if i < 1 || i > 24 {
		http.Error(w, "tick.num-hours must be 1-24", http.StatusBadRequest)
		return
	} else {
		c.Tick.NumHours = i
	}

	// Tick.StartHour: 0-23 (24h)
	if i, err := strconv.Atoi(r.FormValue("tick.start-hour")); err != nil {
		http.Error(w, "invalid tick.start-hour: "+err.Error(), http.StatusBadRequest)
		return
	} else if i < 0 || i > 23 {
		http.Error(w, "tick.start-hour must be 0-23", http.StatusBadRequest)
		return
	} else {
		c.Tick.StartHour = i
	}

	if v := r.FormValue("tick.past-color"); v != "" {
		color, err := hexStringToUint32(v)
		if err != nil {
			http.Error(w, "tick.past-color: "+err.Error(), http.StatusBadRequest)
			return
		}
		c.Tick.PastColor = color
	}
	if v := r.FormValue("tick.present-color"); v != "" {
		color, err := hexStringToUint32(v)
		if err != nil {
			http.Error(w, "tick.present-color: "+err.Error(), http.StatusBadRequest)
			return
		}
		c.Tick.PresentColor = color
	}
	if v := r.FormValue("tick.future-color"); v != "" {
		color, err := hexStringToUint32(v)
		if err != nil {
			http.Error(w, "tick.future-color: "+err.Error(), http.StatusBadRequest)
			return
		}
		c.Tick.FutureColor = color
	}
	if v := r.FormValue("tick.future-color-b"); v != "" {
		color, err := hexStringToUint32(v)
		if err != nil {
			http.Error(w, "tick.future-color-b: "+err.Error(), http.StatusBadRequest)
			return
		}
		c.Tick.FutureColorB = color
	}
	if v := r.FormValue("num.past-color"); v != "" {
		color, err := hexStringToUint32(v)
		if err != nil {
			http.Error(w, "num.past-color: "+err.Error(), http.StatusBadRequest)
			return
		}
		c.Num.PastColor = color
	}
	if v := r.FormValue("num.present-color"); v != "" {
		color, err := hexStringToUint32(v)
		if err != nil {
			http.Error(w, "num.present-color: "+err.Error(), http.StatusBadRequest)
			return
		}
		c.Num.PresentColor = color
	}
	if v := r.FormValue("num.future-color"); v != "" {
		color, err := hexStringToUint32(v)
		if err != nil {
			http.Error(w, "num.future-color: "+err.Error(), http.StatusBadRequest)
			return
		}
		c.Num.FutureColor = color
	}

	if err := WriteConfigLocked(s.ConfigPath, c); err != nil {
		http.Error(w, "write config: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func hexStringToUint32(hexStr string) (uint32, error) {
	hexStr = strings.TrimPrefix(hexStr, "#")
	if len(hexStr) < 6 {
		return 0, fmt.Errorf("color hex string too short: %q (need 6 hex digits)", hexStr)
	}

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
