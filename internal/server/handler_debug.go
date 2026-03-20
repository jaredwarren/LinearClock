package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/jaredwarren/clock/internal/config"
	"github.com/jaredwarren/clock/internal/display"
)

// DebugEventsResponse is the JSON response for GET /debug/events.
type DebugEventsResponse struct {
	Now             string                 `json:"now"`
	NowTimezone     string                 `json:"now_timezone"`
	ConfigPath      string                 `json:"config_path"`
	ConfigLoaded    bool                   `json:"config_loaded"`
	EventsInConfig  int                    `json:"events_in_config"`
	Events          []DebugEventEntry      `json:"events"`
	BaseColors      DebugColors            `json:"base_colors"`
	EffectiveColors DebugColors            `json:"effective_colors"`
}

type DebugEventEntry struct {
	Title    string     `json:"title"`
	Start    string     `json:"start"`
	End      string     `json:"end"`
	Repeat   string     `json:"repeat"`
	Matches  bool       `json:"matches"`
	Overrides DebugColors `json:"overrides"`
}

type DebugColors struct {
	Past    string `json:"past"`
	Present string `json:"present"`
	Future  string `json:"future"`
	FutureB string `json:"future_b"`
}

func (s *Server) DebugEvents(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	if to := r.URL.Query().Get("time"); to != "" {
		if t, err := time.Parse("2006-01-02T15:04", to); err == nil {
			now = t
		}
	}

	c, err := config.ReadConfig(s.ConfigPath)
	loaded := err == nil
	if err != nil && errorIsMissing(err) {
		c = config.DefaultConfig.Clone()
		loaded = true
	}

	resp := DebugEventsResponse{
		Now:            now.Format(time.RFC3339),
		NowTimezone:    now.Location().String(),
		ConfigPath:     s.ConfigPath,
		ConfigLoaded:   loaded,
		EventsInConfig: 0,
		Events:         nil,
		BaseColors: DebugColors{
			Past:    "#000000",
			Present: "#000000",
			Future:  "#000000",
			FutureB: "#000000",
		},
		EffectiveColors: DebugColors{
			Past:    "#000000",
			Present: "#000000",
			Future:  "#000000",
			FutureB: "#000000",
		},
	}

	if c != nil {
		resp.BaseColors = DebugColors{
			Past:    ColorString(c.Tick.PastColor),
			Present: ColorString(c.Tick.PresentColor),
			Future:  ColorString(c.Tick.FutureColor),
			FutureB: ColorString(c.Tick.FutureColorB),
		}
		effective := display.ResolveTickColorsForTime(now, c.Tick, c.Tick.Events)
		resp.EffectiveColors = DebugColors{
			Past:    ColorString(effective.Past),
			Present: ColorString(effective.Present),
			Future:  ColorString(effective.Future),
			FutureB: ColorString(effective.FutureB),
		}
		if c.Tick.Events != nil {
			resp.EventsInConfig = len(c.Tick.Events)
			resp.Events = make([]DebugEventEntry, len(c.Tick.Events))
			for i := range c.Tick.Events {
				e := &c.Tick.Events[i]
				resp.Events[i] = DebugEventEntry{
					Title:   e.Title,
					Start:   e.Start.Format(time.RFC3339),
					End:     e.End.Format(time.RFC3339),
					Repeat:  e.Repeat,
					Matches: display.EventMatches(now, e),
					Overrides: DebugColors{
						Past:    ColorString(e.PastColorOverride),
						Present: ColorString(e.PresentColorOverride),
						Future:  ColorString(e.FutureColorOverride),
						FutureB: ColorString(e.FutureColorBOverride),
					},
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(resp)
}
