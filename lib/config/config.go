package config

import "time"

type Config struct {
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
	StartHour    int // 24h time
	StartLed     int
	Reverse      bool // increment up or down each tick
	TicksPerHour int
	NumHours     int

	// V2 config features to add
	Mode string // "count down", "count up", "time", etc
}

// NumConfig ...
type NumConfig struct {
	NumLeds int
	Reverse bool   // increment up or down with time
	Mode    string // "count down", "count up", "time", etc
}
