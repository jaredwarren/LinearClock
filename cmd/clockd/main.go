package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jaredwarren/clock/internal/config"
	"github.com/jaredwarren/clock/internal/display"
	"github.com/jaredwarren/clock/internal/hw"
)

// DefaultConfigFile is the default path to the configuration file.
const DefaultConfigFile = "config.gob"

var defaultCfg = &config.Config{
	RefreshRate: 1 * time.Minute,
	Brightness:  64,
	Tick: config.TickConfig{
		FutureColor:  0x00ff00,
		PastColor:    0xFF0000,
		PresentColor: 0,
		StartHour:    6,
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

func main() {
	configFile := flag.String("config", DefaultConfigFile, "Path to configuration file.")
	flag.Parse()

	log.Println("starting...")
	c, err := config.ReadConfig(*configFile)
	if err != nil {
		log.Printf("read config error, using default: %v", err)
		c = defaultCfg
	}

	dev, err := hw.NewLedDisplay(c)
	if err != nil {
		log.Fatalf("failed to create display: %v", err)
	}

	if err := dev.Init(); err != nil {
		log.Fatalf("failed to initialize display: %v", err)
	}
	defer dev.Fini()

	time.Sleep(500 * time.Millisecond)
	display.Clear(dev)
	time.Sleep(500 * time.Millisecond)

	ctx, stop := context.WithCancel(context.Background())
	defer stop()

	go startClock(ctx, dev, c, *configFile)

	log.Println("running...")
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Print("shutting down...")
	stop()

	display.Clear(dev)
	time.Sleep(1 * time.Second)
	log.Println("done")
}

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
				return
			}

			if nc, err := config.ReadConfig(configFile); err == nil {
				if nc.RefreshRate != cfg.RefreshRate {
					ticker.Reset(nc.RefreshRate)
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
