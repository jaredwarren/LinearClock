package config

import (
	"encoding/gob"
	"fmt"
	"os"
	"time"
)

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
	Version:     1,
	RefreshRate: 1 * time.Minute,
	Brightness:  128,
	Tick: TickConfig{
		FutureColor:  0x00ff00,
		PastColor:    0xFF0000,
		StartHour:    8, // for testing set to curren thour or lower
		StartLed:     1,
		Reverse:      false,
		TicksPerHour: 4,
		NumHours:     6,
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

	return &config, nil
}

// Clone returns a deep copy of c. Returns nil if c is nil.
func (c *Config) Clone() *Config {
	if c == nil {
		return nil
	}
	return &Config{
		Version:     c.Version,
		DisplayMode: c.DisplayMode,
		Brightness:  c.Brightness,
		RefreshRate: c.RefreshRate,
		Tick:        c.Tick,
		Num:         c.Num,
		Gap:         c.Gap,
	}
}

func WriteConfig(filepath string, c *Config) error {
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("create config file: %w", err)
	}
	defer file.Close()

	enc := gob.NewEncoder(file)
	err = enc.Encode(c)
	if err != nil {
		return fmt.Errorf("encode config %s - %w", filepath, err)
	}
	return nil
}
