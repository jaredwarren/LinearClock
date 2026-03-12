package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
