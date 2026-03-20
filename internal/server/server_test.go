package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jaredwarren/clock/internal/config"
)

func TestHexStringToUint32(t *testing.T) {
	tests := []struct {
		name    string
		hexStr  string
		want    uint32
		wantErr bool
	}{
		{"6 hex digits", "FF0000", 0xFF0000, false},
		{"with hash", "#00FF00", 0x00FF00, false},
		{"lowercase", "0000ff", 0x0000FF, false},
		{"black", "000000", 0, false},
		{"white", "FFFFFF", 0xFFFFFF, false},
		{"empty", "", 0, true},
		{"short", "FF", 0, true},
		{"five chars", "12345", 0, true},
		{"with hash short", "#ab", 0, true},
		{"invalid hex", "GGGGGG", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := hexStringToUint32(tt.hexStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("hexStringToUint32(%q) err = %v, wantErr %v", tt.hexStr, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("hexStringToUint32(%q) = %#x, want %#x", tt.hexStr, got, tt.want)
			}
		})
	}
}

func TestUpdateConfig_InvalidBrightness_Returns400(t *testing.T) {
	s := New(t.TempDir() + "/config.gob")
	body := "brightness=999&refresh-rate=60&gap=2&tick.start-led=0&tick.ticks-per-hour=4&tick.num-hours=6&tick.start-hour=9"
	req := httptest.NewRequest(http.MethodPost, "/config", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	s.UpdateConfig(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("UpdateConfig(brightness=999) status = %d, want 400", rec.Code)
	}
}

func TestUpdateConfig_InvalidRefreshRate_Returns400(t *testing.T) {
	s := New(t.TempDir() + "/config.gob")
	body := "brightness=128&refresh-rate=0&gap=2&tick.start-led=0&tick.ticks-per-hour=4&tick.num-hours=6&tick.start-hour=9"
	req := httptest.NewRequest(http.MethodPost, "/config", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	s.UpdateConfig(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("UpdateConfig(refresh-rate=0) status = %d, want 400", rec.Code)
	}
}

func TestUpdateConfig_InvalidColor_Returns400(t *testing.T) {
	s := New(t.TempDir() + "/config.gob")
	body := "brightness=128&refresh-rate=60&gap=2&tick.start-led=0&tick.ticks-per-hour=4&tick.num-hours=6&tick.start-hour=9&tick.past-color=x"
	req := httptest.NewRequest(http.MethodPost, "/config", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	s.UpdateConfig(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("UpdateConfig(tick.past-color=x) status = %d, want 400", rec.Code)
	}
}

func TestUpdateConfig_ValidForm_Redirects(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/config.gob"
	s := New(path)

	body := "brightness=128&refresh-rate=60&gap=2&tick.start-led=0&tick.ticks-per-hour=4&tick.num-hours=6&tick.start-hour=9"
	req := httptest.NewRequest(http.MethodPost, "/config", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	s.UpdateConfig(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Errorf("UpdateConfig(valid) status = %d, want 303", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/" {
		t.Errorf("Location = %q, want \"/\"", loc)
	}
}

func TestUpdateEvents_ValidForm_PersistsAndRedirects(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/config.gob"
	s := New(path)

	body := "event.title=Meeting&event.start=2024-06-15T10:00&event.end=2024-06-15T11:00&event.repeat=none&event.past-color=%23FF0000"
	req := httptest.NewRequest(http.MethodPost, "/events", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	s.UpdateEvents(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Errorf("UpdateEvents status = %d, want 303", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/events" {
		t.Errorf("Location = %q, want \"/events\"", loc)
	}
	c, err := config.ReadConfig(path)
	if err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}
	if len(c.Tick.Events) != 1 {
		t.Fatalf("len(Events) = %d, want 1", len(c.Tick.Events))
	}
	e := c.Tick.Events[0]
	if e.Title != "Meeting" {
		t.Errorf("Title = %q, want Meeting", e.Title)
	}
	if e.Repeat != config.RepeatNone {
		t.Errorf("Repeat = %q, want none", e.Repeat)
	}
	if e.PastColorOverride != 0xFF0000 {
		t.Errorf("PastColorOverride = %#x, want 0xFF0000", e.PastColorOverride)
	}
	if e.ID == "" {
		t.Error("ID should be set")
	}
}

func TestDeleteEventPost_RemovesEventAndRedirects(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/config.gob"
	start, _ := time.Parse("2006-01-02T15:04", "2024-06-15T10:00")
	end, _ := time.Parse("2006-01-02T15:04", "2024-06-15T11:00")
	c := config.DefaultConfig.Clone()
	c.Tick.Events = []config.TickEvent{
		{ID: "evt_abc", Title: "X", Start: start, End: end, Repeat: config.RepeatNone},
	}
	if err := config.WriteConfig(path, c); err != nil {
		t.Fatalf("WriteConfig: %v", err)
	}
	s := New(path)

	req := httptest.NewRequest(http.MethodPost, "/events/delete", strings.NewReader("event_id=evt_abc"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	s.DeleteEventPost(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Errorf("DeleteEventPost status = %d, want 303", rec.Code)
	}
	c2, err := config.ReadConfig(path)
	if err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}
	if len(c2.Tick.Events) != 0 {
		t.Errorf("len(Events) = %d, want 0", len(c2.Tick.Events))
	}
}

func TestMoveEventUp_SwapsOrderAndRedirects(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/config.gob"
	start, _ := time.Parse("2006-01-02T15:04", "2024-06-15T10:00")
	end, _ := time.Parse("2006-01-02T15:04", "2024-06-15T11:00")
	c := config.DefaultConfig.Clone()
	c.Tick.Events = []config.TickEvent{
		{ID: "first", Title: "First", Start: start, End: end, Repeat: config.RepeatNone},
		{ID: "second", Title: "Second", Start: start, End: end, Repeat: config.RepeatNone},
	}
	if err := config.WriteConfig(path, c); err != nil {
		t.Fatalf("WriteConfig: %v", err)
	}
	s := New(path)

	req := httptest.NewRequest(http.MethodPost, "/events/move-up", strings.NewReader("index=1"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	s.MoveEventUp(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Errorf("MoveEventUp status = %d, want 303", rec.Code)
	}
	c2, err := config.ReadConfig(path)
	if err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}
	if len(c2.Tick.Events) != 2 {
		t.Fatalf("len(Events) = %d, want 2", len(c2.Tick.Events))
	}
	if c2.Tick.Events[0].ID != "second" || c2.Tick.Events[1].ID != "first" {
		t.Errorf("order after move-up: got [%q, %q], want [second, first]", c2.Tick.Events[0].ID, c2.Tick.Events[1].ID)
	}
}

func TestMoveEventDown_SwapsOrderAndRedirects(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/config.gob"
	start, _ := time.Parse("2006-01-02T15:04", "2024-06-15T10:00")
	end, _ := time.Parse("2006-01-02T15:04", "2024-06-15T11:00")
	c := config.DefaultConfig.Clone()
	c.Tick.Events = []config.TickEvent{
		{ID: "first", Title: "First", Start: start, End: end, Repeat: config.RepeatNone},
		{ID: "second", Title: "Second", Start: start, End: end, Repeat: config.RepeatNone},
	}
	if err := config.WriteConfig(path, c); err != nil {
		t.Fatalf("WriteConfig: %v", err)
	}
	s := New(path)

	req := httptest.NewRequest(http.MethodPost, "/events/move-down", strings.NewReader("index=0"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	s.MoveEventDown(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Errorf("MoveEventDown status = %d, want 303", rec.Code)
	}
	c2, err := config.ReadConfig(path)
	if err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}
	if len(c2.Tick.Events) != 2 {
		t.Fatalf("len(Events) = %d, want 2", len(c2.Tick.Events))
	}
	if c2.Tick.Events[0].ID != "second" || c2.Tick.Events[1].ID != "first" {
		t.Errorf("order after move-down: got [%q, %q], want [second, first]", c2.Tick.Events[0].ID, c2.Tick.Events[1].ID)
	}
}

func TestUpdateEvents_InvalidInput_Returns400(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/config.gob"
	s := New(path)

	tests := []struct {
		name string
		body string
	}{
		{
			name: "invalid start",
			body: "event.title=Bad&event.start=not-a-time&event.end=2024-06-15T11:00&event.repeat=none",
		},
		{
			name: "invalid end",
			body: "event.title=Bad&event.start=2024-06-15T10:00&event.end=not-a-time&event.repeat=none",
		},
		{
			name: "end before start",
			body: "event.title=Bad&event.start=2024-06-15T11:00&event.end=2024-06-15T10:00&event.repeat=none",
		},
		{
			name: "invalid color",
			body: "event.title=Bad&event.start=2024-06-15T10:00&event.end=2024-06-15T11:00&event.repeat=none&event.past-color=x",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/events", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rec := httptest.NewRecorder()

			s.UpdateEvents(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("UpdateEvents(%s) status = %d, want 400", tt.name, rec.Code)
			}
		})
	}
}

func TestEditEvent_TableDriven(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/config.gob"
	start, _ := time.Parse("2006-01-02T15:04", "2024-06-15T10:00")
	end, _ := time.Parse("2006-01-02T15:04", "2024-06-15T11:00")

	base := config.DefaultConfig.Clone()
	base.Tick.Events = []config.TickEvent{
		{
			ID:     "evt_1",
			Title:  "Original",
			Start:  start,
			End:    end,
			Repeat: config.RepeatNone,
		},
	}
	if err := config.WriteConfig(path, base); err != nil {
		t.Fatalf("WriteConfig: %v", err)
	}

	s := New(path)

	tests := []struct {
		name       string
		body       string
		wantStatus int
		check      func(t *testing.T, cfg *config.Config)
	}{
		{
			name:       "valid edit updates event",
			body:       "event_id=evt_1&event.title=Updated&event.start=2024-06-15T12:00&event.end=2024-06-15T13:00&event.repeat=daily&event.past-color=%2300FF00",
			wantStatus: http.StatusSeeOther,
			check: func(t *testing.T, cfg *config.Config) {
				if len(cfg.Tick.Events) != 1 {
					t.Fatalf("len(Events) = %d, want 1", len(cfg.Tick.Events))
				}
				e := cfg.Tick.Events[0]
				if e.Title != "Updated" {
					t.Errorf("Title = %q, want Updated", e.Title)
				}
				if e.Repeat != config.RepeatDaily {
					t.Errorf("Repeat = %q, want daily", e.Repeat)
				}
				if e.PastColorOverride != 0x00FF00 {
					t.Errorf("PastColorOverride = %#x, want 0x00FF00", e.PastColorOverride)
				}
				if e.Start.Hour() != 12 || e.End.Hour() != 13 {
					t.Errorf("Start/End hours = %d/%d, want 12/13", e.Start.Hour(), e.End.Hour())
				}
			},
		},
		{
			name:       "missing id returns 400",
			body:       "event.title=X",
			wantStatus: http.StatusBadRequest,
			check:      nil,
		},
		{
			name:       "bad start returns 400",
			body:       "event_id=evt_1&event.title=X&event.start=bad&event.end=2024-06-15T13:00&event.repeat=none",
			wantStatus: http.StatusBadRequest,
			check:      nil,
		},
		{
			name:       "bad color returns 400",
			body:       "event_id=evt_1&event.title=X&event.start=2024-06-15T12:00&event.end=2024-06-15T13:00&event.repeat=none&event.past-color=x",
			wantStatus: http.StatusBadRequest,
			check:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/events/edit", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rec := httptest.NewRecorder()

			s.EditEvent(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("EditEvent status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.check != nil {
				cfg, err := config.ReadConfig(path)
				if err != nil {
					t.Fatalf("ReadConfig: %v", err)
				}
				tt.check(t, cfg)
			}
		})
	}
}

