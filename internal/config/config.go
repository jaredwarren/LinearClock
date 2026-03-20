package config

import (
	"encoding/gob"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const CurrentConfigVersion = 3

type Config struct {
	Version     int // config schema version for future migrations
	DisplayMode string
	// general
	Brightness  int
	RefreshRate time.Duration

	Tick TickConfig
	Num  NumConfig

	// numbers
	Gap int // number of leds between ticks and numbers

	Calendar CalendarConfig
}

// CalendarConfig controls periodic iCalendar (iCal/ICS) syncing.
// When ICalURL is set, configd periodically fetches the feed, expands recurring
// VEVENTs into concrete occurrences, and replaces Tick.Events with generated events.
type CalendarConfig struct {
	ICalURL string

	// PollIntervalSeconds controls how often configd fetches the iCal feed.
	PollIntervalSeconds int

	// LookbackDays / LookaheadDays define the time window used when expanding
	// recurring events. Generated events whose occurrences fall outside this
	// window are not included in Tick.Events.
	LookbackDays  int
	LookaheadDays int

	// LED override colors applied to every generated iCal event.
	// 0 means "no override" (fall back to Tick/PastColor/PastColor etc).
	OverridePastColor    uint32
	OverridePresentColor uint32
	OverrideFutureColor  uint32
	OverrideFutureBColor uint32
}

// RepeatNone and RepeatDaily are values for TickEvent.Repeat.
const (
	RepeatNone  = "none"
	RepeatDaily = "daily"
)

// TickEvent is an event that overrides tick colors between Start and End.
// Order in the slice matters: later events override earlier ones.
// Zero value for a color override means "do not override".
type TickEvent struct {
	ID                   string    // stable id for UI (e.g. delete, reorder)
	Title                string    // metadata for display
	Start                time.Time // inclusive
	End                  time.Time // inclusive
	Repeat               string    // RepeatNone (one-time) or RepeatDaily
	PastColorOverride    uint32    // 0 = no override
	PresentColorOverride uint32
	FutureColorOverride  uint32
	FutureColorBOverride uint32
}

// TickConfig ...
type TickConfig struct {
	PastColor    uint32
	PresentColor uint32
	FutureColor  uint32
	FutureColorB uint32

	// Hardware specific configuration
	StartHour    int // 24h time
	StartLed     int
	TicksPerHour int
	NumHours     int

	// V2 config features to add
	Mode    string // "count down", "count up", "time", etc
	Reverse bool   // increment up or down each tick

	// Optional transition rendering between updates.
	TransitionEnabled    bool
	TransitionDurationMs int // total transition time in milliseconds
	TransitionMaxSteps   int // safety cap for interpolation steps

	// Events override tick colors in order when active
	Events []TickEvent
}

// NumConfig ...
type NumConfig struct {
	PastColor    uint32
	PresentColor uint32
	FutureColor  uint32
	// V2 config features to add
	Reverse bool   // increment up or down with time
	Mode    string // "count down", "count up", "time", etc
}

var DefaultConfig = &Config{
	Version:     CurrentConfigVersion,
	RefreshRate: 1 * time.Minute,
	Brightness:  128,
	Calendar: CalendarConfig{
		ICalURL:             "",
		PollIntervalSeconds: 300, // 5 minutes
		LookbackDays:        1,
		LookaheadDays:       14,
		// Default overrides are intentionally distinct to make iCal highlighting visible.
		OverridePastColor:     0xFF00FF,
		OverridePresentColor:  0x00FFFF,
		OverrideFutureColor:   0xFFFF00,
		OverrideFutureBColor:  0x00FF00,
	},
	Tick: TickConfig{
		FutureColor:  0x00ff00,
		PastColor:    0xFF0000,
		StartHour:    8, // for testing set to curren thour or lower
		StartLed:     1,
		Reverse:      false,
		TicksPerHour: 4,
		NumHours:     6,
		// Disabled by default for low-overhead behavior.
		TransitionEnabled:    false,
		TransitionDurationMs: 0,
		TransitionMaxSteps:   6,
	},
	Num: NumConfig{
		PastColor:    0xffff00,
		FutureColor:  0x00ffff,
		PresentColor: 0xff00ff,
	},
	Gap: 4,
}

func ReadConfig(filepath string) (*Config, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := gob.NewDecoder(file)
	var config Config
	err = decoder.Decode(&config)
	if err != nil {
		return nil, err
	}

	config.Migrate()
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &config, nil
}

// Clone returns a deep copy of c. Returns nil if c is nil.
func (c *Config) Clone() *Config {
	if c == nil {
		return nil
	}
	tick := c.Tick
	if n := len(c.Tick.Events); n > 0 {
		tick.Events = make([]TickEvent, n)
		copy(tick.Events, c.Tick.Events)
	} else {
		tick.Events = nil
	}
	return &Config{
		Version:     c.Version,
		DisplayMode: c.DisplayMode,
		Brightness:  c.Brightness,
		RefreshRate: c.RefreshRate,
		Tick:        tick,
		Num:         c.Num,
		Gap:         c.Gap,
		Calendar:    c.Calendar,
	}
}

func WriteConfig(filepath string, c *Config) error {
	if c == nil {
		return errors.New("config is nil")
	}
	c = c.Clone()
	c.Migrate()
	if err := c.Validate(); err != nil {
		return fmt.Errorf("validate config: %w", err)
	}

	// Write atomically so readers won't observe partial gob data.
	// This matters once configd starts a periodic sync loop that overwrites Tick.Events.
	tmpPath := filepath + ".tmp"

	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("create temp config file: %w", err)
	}

	enc := gob.NewEncoder(f)
	if err := enc.Encode(c); err != nil {
		_ = f.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("encode config %s - %w", filepath, err)
	}

	// Best-effort flush.
	_ = f.Sync()
	_ = f.Close()

	if err := os.Rename(tmpPath, filepath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename temp config file: %w", err)
	}

	// Keep a last-known-good backup for startup rollback.
	_ = copyFile(filepath, backupPath(filepath))

	return nil
}

func backupPath(path string) string {
	return path + ".bak"
}

// TryReadWithBackupRollback reads config from path. If decode/validation fails,
// it attempts to restore from path+".bak" and re-read.
func TryReadWithBackupRollback(path string) (*Config, error) {
	c, err := ReadConfig(path)
	if err == nil {
		return c, nil
	}
	if restoreErr := RestoreBackup(path); restoreErr != nil {
		return nil, fmt.Errorf("read config failed: %w; restore backup failed: %v", err, restoreErr)
	}
	return ReadConfig(path)
}

// RestoreBackup restores path from path+".bak".
func RestoreBackup(path string) error {
	bak := backupPath(path)
	if _, err := os.Stat(bak); err != nil {
		return fmt.Errorf("backup missing: %w", err)
	}
	if err := copyFile(bak, path); err != nil {
		return fmt.Errorf("copy backup to config: %w", err)
	}
	return nil
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}

// Migrate applies schema defaults/adjustments for older config versions.
func (c *Config) Migrate() {
	if c == nil {
		return
	}
	if c.Version <= 0 {
		c.Version = 1
	}

	// Common defaults that keep old gobs operational.
	if c.RefreshRate <= 0 {
		c.RefreshRate = DefaultConfig.RefreshRate
	}
	if c.Tick.TicksPerHour <= 0 {
		c.Tick.TicksPerHour = DefaultConfig.Tick.TicksPerHour
	}
	if c.Tick.NumHours <= 0 {
		c.Tick.NumHours = DefaultConfig.Tick.NumHours
	}
	if c.Calendar.PollIntervalSeconds <= 0 {
		c.Calendar.PollIntervalSeconds = DefaultConfig.Calendar.PollIntervalSeconds
	}
	if c.Calendar.LookbackDays < 0 {
		c.Calendar.LookbackDays = DefaultConfig.Calendar.LookbackDays
	}
	if c.Calendar.LookaheadDays < 0 {
		c.Calendar.LookaheadDays = DefaultConfig.Calendar.LookaheadDays
	}
	if c.Tick.TransitionMaxSteps <= 0 {
		c.Tick.TransitionMaxSteps = DefaultConfig.Tick.TransitionMaxSteps
	}
	if c.Tick.TransitionDurationMs < 0 {
		c.Tick.TransitionDurationMs = 0
	}

	if c.Version < CurrentConfigVersion {
		c.Version = CurrentConfigVersion
	}
}

// Validate centralizes config validation for handlers/startup/sync loops.
func (c *Config) Validate() error {
	if c == nil {
		return errors.New("config is nil")
	}
	if c.Brightness < 0 || c.Brightness > 256 {
		return fmt.Errorf("brightness must be 0-256")
	}
	if c.RefreshRate < time.Second || c.RefreshRate > 900*time.Second {
		return fmt.Errorf("refresh-rate must be 1-900 seconds")
	}
	if c.Gap < 0 || c.Gap > 100 {
		return fmt.Errorf("gap must be 0-100")
	}
	if c.Tick.StartLed < 0 {
		return fmt.Errorf("tick.start-led must be non-negative")
	}
	if c.Tick.TicksPerHour < 1 || c.Tick.TicksPerHour > 60 {
		return fmt.Errorf("tick.ticks-per-hour must be 1-60")
	}
	if c.Tick.NumHours < 1 || c.Tick.NumHours > 24 {
		return fmt.Errorf("tick.num-hours must be 1-24")
	}
	if c.Tick.StartHour < 0 || c.Tick.StartHour > 23 {
		return fmt.Errorf("tick.start-hour must be 0-23")
	}
	if c.Calendar.PollIntervalSeconds < 10 || c.Calendar.PollIntervalSeconds > 86400 {
		return fmt.Errorf("calendar.poll-interval-seconds must be 10-86400")
	}
	if c.Calendar.LookbackDays < 0 || c.Calendar.LookbackDays > 365 {
		return fmt.Errorf("calendar.lookback-days must be 0-365")
	}
	if c.Calendar.LookaheadDays < 0 || c.Calendar.LookaheadDays > 365 {
		return fmt.Errorf("calendar.lookahead-days must be 0-365")
	}
	if c.Tick.TransitionDurationMs < 0 || c.Tick.TransitionDurationMs > 2000 {
		return fmt.Errorf("tick.transition-duration-ms must be 0-2000")
	}
	if c.Tick.TransitionMaxSteps < 1 || c.Tick.TransitionMaxSteps > 12 {
		return fmt.Errorf("tick.transition-max-steps must be 1-12")
	}

	for i := range c.Tick.Events {
		e := c.Tick.Events[i]
		if e.End.Before(e.Start) {
			return fmt.Errorf("event[%d]: end before start", i)
		}
		if e.Repeat != RepeatNone && e.Repeat != RepeatDaily {
			return fmt.Errorf("event[%d]: invalid repeat %q", i, e.Repeat)
		}
	}

	return nil
}
