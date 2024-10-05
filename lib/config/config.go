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
	OnColor      uint32
	OffColor     uint32
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
	OnColor  uint32
	OffColor uint32
	// V2 config features to add
	Reverse bool   // increment up or down with time
	Mode    string // "count down", "count up", "time", etc
}
