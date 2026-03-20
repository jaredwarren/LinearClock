package main

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jaredwarren/clock/internal/calendar"
	"github.com/jaredwarren/clock/internal/config"
	"github.com/jaredwarren/clock/internal/server"
)

const defaultConfigPath = "config.gob"

func main() {
	s := server.New(defaultConfigPath)
	if _, err := config.TryReadWithBackupRollback(s.ConfigPath); err != nil {
		log.Printf("startup config read/rollback failed; proceeding with defaults until config is repaired: %v", err)
	}

	ctx := context.Background()
	go func() {
		for {
			// Reload config each cycle so poll interval / lookback / lookahead changes apply immediately.
			c, err := config.TryReadWithBackupRollback(s.ConfigPath)
			if err != nil {
				c = config.DefaultConfig.Clone()
			}

			// Only sync when configured.
			if strings.TrimSpace(c.Calendar.ICalURL) != "" {
				events, err := calendar.SyncICalToTickEvents(ctx, c, time.Now())
				if err == nil {
					c.Tick.Events = events
					if err := c.Validate(); err == nil {
						_ = server.WriteConfigLocked(s.ConfigPath, c)
					}
				}
			}

			intervalSeconds := c.Calendar.PollIntervalSeconds
			if intervalSeconds < 10 {
				intervalSeconds = 300
			}
			time.Sleep(time.Duration(intervalSeconds) * time.Second)
		}
	}()

	http.HandleFunc("GET /", s.Home)
	http.HandleFunc("GET /events", s.ListEvents)
	http.HandleFunc("POST /events", s.UpdateEvents)
	http.HandleFunc("POST /events/move-up", s.MoveEventUp)
	http.HandleFunc("POST /events/move-down", s.MoveEventDown)
	http.HandleFunc("POST /events/delete", s.DeleteEventPost)
	http.HandleFunc("POST /events/edit", s.EditEvent)
	http.HandleFunc("DELETE /event/{event_id}", s.DeleteEvent)
	http.HandleFunc("GET /test", s.TestHandler)
	http.HandleFunc("GET /debug/events", s.DebugEvents)
	http.HandleFunc("POST /config", s.UpdateConfig)
	http.Handle("GET /public/", http.StripPrefix("/public/", http.FileServer(http.Dir("./public"))))

	log.Println("Server listening on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("server: %v", err)
	}
}
