package config

import (
	"encoding/gob"
	"os"
	"time"
)

type Config struct {
	// general
	Brightness  int
	RefreshRate time.Duration

	Tick TickConfig
	Num  NumConfig

	// numbers
	Gap int // number of leds between ticks and numbers

	// V2
	DisplayMode string
}

// TickConfig ...
type TickConfig struct {
	PastColor    uint32
	PresentColor uint32
	FutureColor  uint32
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

func ReadConfig(filepath string) (*Config, error) {
	// Open the file for reading
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Create a new decoder
	decoder := gob.NewDecoder(file)

	// Decode the data
	var config Config
	err = decoder.Decode(&config)
	if err != nil {
		return nil, err
	}

	// Print the decoded data
	return &config, nil
}
