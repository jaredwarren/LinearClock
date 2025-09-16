package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jaredwarren/clock/clock/config"
	"github.com/jaredwarren/clock/clock/display"
	ws2811 "github.com/rpi-ws281x/rpi-ws281x-go"
)

// DefaultConfigFile is the default path to the configuration file.
const DefaultConfigFile = "config.gob"

var DefaultConfig = &config.Config{
	RefreshRate: 1 * time.Minute,
	Brightness:  64,
	Tick: config.TickConfig{
		FutureColor:  0x00ff00,
		PastColor:    0xFF0000,
		PresentColor: 0,
		StartHour:    6, // for testing set to curren thour or lower
		StartLed:     0,
		Reverse:      false,
		TicksPerHour: 4,
		NumHours:     18,
	},
	Num: config.NumConfig{
		PastColor:    0xff0000,
		FutureColor:  0x00ff00,
		PresentColor: 0x0000ff,
	},
	Gap: 0,
}

// NewLedDisplay initializes the LED hardware with settings from the config.
func NewLedDisplay(c *config.Config) (*ws2811.WS2811, error) {
	opt := ws2811.DefaultOptions
	opt.Channels[0].Brightness = c.Brightness

	// Dynamically calculate the number of LEDs required.
	numLeds := (c.Tick.NumHours * c.Tick.TicksPerHour) * 2
	opt.Channels[0].LedCount = numLeds
	return ws2811.MakeWS2811(&opt)
}

func main() {
	configFile := flag.String("config", DefaultConfigFile, "Path to configuration file.")
	flag.Parse()

	log.Println("starting...")
	c, err := config.ReadConfig(*configFile)
	if err != nil {
		log.Printf("read config error, using default: %v", err)
		c = DefaultConfig
	}

	dev, err := NewLedDisplay(c)
	if err != nil {
		log.Fatalf("failed to create display: %v", err)
	}

	if err := dev.Init(); err != nil {
		log.Fatalf("failed to initialize display: %v", err)
	}
	defer dev.Fini()

	// A brief startup sequence to clear the display.
	time.Sleep(500 * time.Millisecond)
	display.Clear(dev)
	time.Sleep(500 * time.Millisecond)

	// Set up a context for graceful shutdown.
	ctx, stop := context.WithCancel(context.Background())
	defer stop()

	// Start the clock in a separate goroutine.
	go startClock(ctx, dev, c, *configFile)

	// Wait for an interrupt signal to gracefully shut down.
	log.Println("running...")
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Print("shutting down...")
	stop() // Signal the clock goroutine to stop.

	// Clean up by clearing the display.
	display.Clear(dev)
	time.Sleep(1 * time.Second)
	log.Println("done")
}

// startClock runs the main loop for the clock, updating the display periodically.
func startClock(ctx context.Context, dev display.Displayer, initialCfg *config.Config, configFile string) {
	if dev == nil {
		return
	}

	cfg := initialCfg
	ticker := time.NewTicker(cfg.RefreshRate)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Println("tick:", time.Now())
			if err := display.DisplayTime(time.Now(), cfg, dev); err != nil {
				log.Printf("display time error: %v", err)
				return // Exit if display fails.
			}

			// Periodically check for configuration changes.
			if nc, err := config.ReadConfig(configFile); err == nil {
				if nc.RefreshRate != cfg.RefreshRate {
					ticker.Reset(nc.RefreshRate) // Update ticker if refresh rate changed.
				}
				cfg = nc
			} else {
				log.Printf("could not refresh config, using old: %v", err)
			}
		case <-ctx.Done():
			log.Println("stopping clock.")
			return
		}
	}
}
