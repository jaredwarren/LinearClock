package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadConfig_MissingFile(t *testing.T) {
	_, err := ReadConfig(filepath.Join(t.TempDir(), "nonexistent.gob"))
	if err == nil {
		t.Fatal("ReadConfig: expected error for missing file")
	}
	if !os.IsNotExist(err) {
		t.Errorf("ReadConfig: expected os.IsNotExist, got %v", err)
	}
}

func TestReadConfig_InvalidContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.gob")
	if err := os.WriteFile(path, []byte("not gob data"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := ReadConfig(path)
	if err == nil {
		t.Fatal("ReadConfig: expected error for invalid gob content")
	}
}

func TestWriteConfig_ThenRead(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.gob")

	c := DefaultConfig.Clone()
	c.Brightness = 200
	c.Gap = 5

	if err := WriteConfig(path, c); err != nil {
		t.Fatalf("WriteConfig: %v", err)
	}

	got, err := ReadConfig(path)
	if err != nil {
		t.Fatalf("ReadConfig after write: %v", err)
	}
	if got.Brightness != 200 {
		t.Errorf("Brightness = %d, want 200", got.Brightness)
	}
	if got.Gap != 5 {
		t.Errorf("Gap = %d, want 5", got.Gap)
	}
}

func TestWriteConfig_CreateFails(t *testing.T) {
	// Write to a path under a non-existent directory so Create fails
	path := filepath.Join(t.TempDir(), "subdir", "config.gob")
	c := DefaultConfig.Clone()
	err := WriteConfig(path, c)
	if err == nil {
		t.Fatal("WriteConfig: expected error when parent dir missing")
	}
}

func TestClone_ReturnsCopy(t *testing.T) {
	orig := DefaultConfig.Clone()
	if orig == nil {
		t.Fatal("Clone: got nil")
	}

	clone := orig.Clone()
	if clone == nil {
		t.Fatal("Clone of clone: got nil")
	}
	if clone == orig {
		t.Fatal("Clone: returned same pointer as original")
	}

	clone.Brightness = 99
	clone.Tick.NumHours = 12
	if orig.Brightness == 99 || orig.Tick.NumHours == 12 {
		t.Error("Clone: modifying clone changed original")
	}
}

func TestClone_NilReceiver(t *testing.T) {
	var c *Config
	got := c.Clone()
	if got != nil {
		t.Errorf("Clone(nil) = %v, want nil", got)
	}
}

func TestClone_DefaultConfigNotMutated(t *testing.T) {
	// Using DefaultConfig.Clone() and mutating the result must not change DefaultConfig
	baseBrightness := DefaultConfig.Brightness
	baseGap := DefaultConfig.Gap

	c := DefaultConfig.Clone()
	c.Brightness = 255
	c.Gap = 10

	if DefaultConfig.Brightness != baseBrightness {
		t.Errorf("DefaultConfig.Brightness mutated: got %d, want %d", DefaultConfig.Brightness, baseBrightness)
	}
	if DefaultConfig.Gap != baseGap {
		t.Errorf("DefaultConfig.Gap mutated: got %d, want %d", DefaultConfig.Gap, baseGap)
	}
}

func TestReadConfig_ValidGob(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.gob")

	want := &Config{
		Version:     1,
		Brightness:  64,
		RefreshRate: 30 * time.Second,
		Gap:         2,
		Tick: TickConfig{
			TicksPerHour: 4,
			NumHours:     6,
			StartHour:    9,
		},
	}
	if err := WriteConfig(path, want); err != nil {
		t.Fatal(err)
	}

	got, err := ReadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Brightness != want.Brightness || got.RefreshRate != want.RefreshRate ||
		got.Gap != want.Gap || got.Tick.TicksPerHour != want.Tick.TicksPerHour ||
		got.Tick.NumHours != want.Tick.NumHours || got.Tick.StartHour != want.Tick.StartHour {
		t.Errorf("ReadConfig: got %+v, want %+v", got, want)
	}
}

func TestWriteConfig_CreatesBackup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.gob")

	c := DefaultConfig.Clone()
	if err := WriteConfig(path, c); err != nil {
		t.Fatalf("WriteConfig: %v", err)
	}
	if _, err := os.Stat(path + ".bak"); err != nil {
		t.Fatalf("expected backup file, got stat error: %v", err)
	}
}

func TestTryReadWithBackupRollback_RestoresCorruptConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.gob")

	orig := DefaultConfig.Clone()
	orig.Brightness = 222
	if err := WriteConfig(path, orig); err != nil {
		t.Fatalf("WriteConfig: %v", err)
	}

	// Corrupt the active config file; backup should still be valid.
	if err := os.WriteFile(path, []byte("corrupt gob"), 0o644); err != nil {
		t.Fatalf("write corrupt config: %v", err)
	}

	got, err := TryReadWithBackupRollback(path)
	if err != nil {
		t.Fatalf("TryReadWithBackupRollback: %v", err)
	}
	if got.Brightness != orig.Brightness {
		t.Fatalf("brightness after rollback = %d, want %d", got.Brightness, orig.Brightness)
	}

	// Ensure the active file is readable again after rollback.
	if _, err := ReadConfig(path); err != nil {
		t.Fatalf("ReadConfig after rollback: %v", err)
	}
}

func TestValidate_BadRanges(t *testing.T) {
	c := DefaultConfig.Clone()
	c.Tick.TicksPerHour = 0
	if err := c.Validate(); err == nil {
		t.Fatal("expected Validate error for tick.ticks-per-hour=0")
	}
}

func TestValidate_TableDriven(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr bool
	}{
		{
			name:    "valid default config",
			mutate:  func(c *Config) {},
			wantErr: false,
		},
		{
			name: "brightness below range",
			mutate: func(c *Config) {
				c.Brightness = -1
			},
			wantErr: true,
		},
		{
			name: "brightness above range",
			mutate: func(c *Config) {
				c.Brightness = 257
			},
			wantErr: true,
		},
		{
			name: "refresh rate too low",
			mutate: func(c *Config) {
				c.RefreshRate = 500 * time.Millisecond
			},
			wantErr: true,
		},
		{
			name: "calendar poll interval too low",
			mutate: func(c *Config) {
				c.Calendar.PollIntervalSeconds = 1
			},
			wantErr: true,
		},
		{
			name: "transition max steps too high",
			mutate: func(c *Config) {
				c.Tick.TransitionMaxSteps = 99
			},
			wantErr: true,
		},
		{
			name: "event end before start",
			mutate: func(c *Config) {
				now := time.Now()
				c.Tick.Events = []TickEvent{
					{
						ID:    "bad",
						Start: now,
						End:   now.Add(-time.Minute),
						Repeat: RepeatNone,
					},
				}
			},
			wantErr: true,
		},
		{
			name: "invalid repeat value",
			mutate: func(c *Config) {
				now := time.Now()
				c.Tick.Events = []TickEvent{
					{
						ID:     "bad-repeat",
						Start:  now,
						End:    now.Add(time.Minute),
						Repeat: "weekly",
					},
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := DefaultConfig.Clone()
			tt.mutate(c)
			err := c.Validate()
			if tt.wantErr && err == nil {
				t.Fatal("expected validation error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no validation error, got: %v", err)
			}
		})
	}
}

func TestMigrate_TableDriven(t *testing.T) {
	tests := []struct {
		name string
		in   Config
		assert func(t *testing.T, got *Config)
	}{
		{
			name: "fills zero defaults and bumps version",
			in: Config{
				Version: 1,
				Tick: TickConfig{
					TransitionDurationMs: -5,
				},
				Calendar: CalendarConfig{
					PollIntervalSeconds: 0,
					LookbackDays:        -1,
					LookaheadDays:       -2,
				},
			},
			assert: func(t *testing.T, got *Config) {
				if got.Version != CurrentConfigVersion {
					t.Fatalf("version = %d, want %d", got.Version, CurrentConfigVersion)
				}
				if got.RefreshRate != DefaultConfig.RefreshRate {
					t.Fatalf("refreshRate = %v, want default %v", got.RefreshRate, DefaultConfig.RefreshRate)
				}
				if got.Tick.TicksPerHour != DefaultConfig.Tick.TicksPerHour {
					t.Fatalf("ticksPerHour = %d, want default %d", got.Tick.TicksPerHour, DefaultConfig.Tick.TicksPerHour)
				}
				if got.Tick.NumHours != DefaultConfig.Tick.NumHours {
					t.Fatalf("numHours = %d, want default %d", got.Tick.NumHours, DefaultConfig.Tick.NumHours)
				}
				if got.Tick.TransitionDurationMs != 0 {
					t.Fatalf("transition duration = %d, want 0", got.Tick.TransitionDurationMs)
				}
				if got.Calendar.PollIntervalSeconds != DefaultConfig.Calendar.PollIntervalSeconds {
					t.Fatalf("pollInterval = %d, want %d", got.Calendar.PollIntervalSeconds, DefaultConfig.Calendar.PollIntervalSeconds)
				}
				if got.Calendar.LookbackDays != DefaultConfig.Calendar.LookbackDays {
					t.Fatalf("lookback = %d, want %d", got.Calendar.LookbackDays, DefaultConfig.Calendar.LookbackDays)
				}
				if got.Calendar.LookaheadDays != DefaultConfig.Calendar.LookaheadDays {
					t.Fatalf("lookahead = %d, want %d", got.Calendar.LookaheadDays, DefaultConfig.Calendar.LookaheadDays)
				}
			},
		},
		{
			name: "preserves valid configured values",
			in: Config{
				Version: 2,
				RefreshRate: 30 * time.Second,
				Tick: TickConfig{
					TicksPerHour: 8,
					NumHours: 10,
					TransitionMaxSteps: 4,
				},
				Calendar: CalendarConfig{
					PollIntervalSeconds: 120,
					LookbackDays: 2,
					LookaheadDays: 3,
				},
			},
			assert: func(t *testing.T, got *Config) {
				if got.RefreshRate != 30*time.Second {
					t.Fatalf("refreshRate changed unexpectedly: %v", got.RefreshRate)
				}
				if got.Tick.TicksPerHour != 8 || got.Tick.NumHours != 10 || got.Tick.TransitionMaxSteps != 4 {
					t.Fatalf("tick fields changed unexpectedly: %+v", got.Tick)
				}
				if got.Calendar.PollIntervalSeconds != 120 || got.Calendar.LookbackDays != 2 || got.Calendar.LookaheadDays != 3 {
					t.Fatalf("calendar fields changed unexpectedly: %+v", got.Calendar)
				}
				if got.Version != CurrentConfigVersion {
					t.Fatalf("version = %d, want %d", got.Version, CurrentConfigVersion)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.in
			got.Migrate()
			tt.assert(t, &got)
		})
	}
}
