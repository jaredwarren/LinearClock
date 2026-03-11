package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jaredwarren/clock/internal/config"
	"github.com/jaredwarren/clock/internal/display"
	"github.com/jaredwarren/clock/lib/mock"
)

const defaultConfigPath = "config.gob"

var defaultConfig = &config.Config{
	Brightness:  300,
	RefreshRate: time.Minute,
	Tick: config.TickConfig{
		FutureColor:  0x00ff00,
		PastColor:    0xFF0000,
		StartHour:    9,
		StartLed:     0,
		Reverse:      false,
		TicksPerHour: 4,
		NumHours:     4,
	},
	Num: config.NumConfig{
		PastColor:    0xff0000,
		FutureColor:  0x00ff00,
		PresentColor: 0x0000ff,
	},
	Gap: 4,
}

func main() {
	cfg, err := config.ReadConfig(defaultConfigPath)
	if err != nil {
		log.Printf("read config error, using default: %v", err)
		cfg = defaultConfig
	}

	numLeds := (cfg.Tick.NumHours*cfg.Tick.TicksPerHour)*2 + cfg.Gap*2
	dev := mock.NewMockDisplay(numLeds, mock.FormatNLine)

	if err := dev.Init(); err != nil {
		log.Fatalf("init mock display: %v", err)
	}
	defer dev.Fini()

	display.Clear(dev)
	time.Sleep(time.Second)

	// Run clock until we receive a shutdown signal.
	done := make(chan struct{})
	go func() {
		startClock(dev, cfg)
		close(done)
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	display.Clear(dev)
	log.Println("cli-clock exiting")
}

func startClock(dev display.Displayer, cfg *config.Config) {
	if dev == nil {
		return
	}

	for {
		if err := display.DisplayTime(time.Now(), cfg, dev); err != nil {
			log.Printf("display time error: %v", err)
		}

		if cfg.RefreshRate > 0 {
			time.Sleep(cfg.RefreshRate)
		} else {
			time.Sleep(2 * time.Second)
		}

		if newCfg, err := config.ReadConfig(defaultConfigPath); err != nil {
			log.Printf("read config, no change: %v", err)
		} else {
			cfg = newCfg
		}
	}
}
