package main

import (
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/jaredwarren/clock/clock/config"
	"github.com/jaredwarren/clock/clock/display"
	ws2811 "github.com/rpi-ws281x/rpi-ws281x-go"
)

const ConfigFile = "config.gob"

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

func NewLedDisplay(c *config.Config) (*ws2811.WS2811, error) {
	opt := ws2811.DefaultOptions
	opt.Channels[0].Brightness = c.Brightness

	// var numLeds = (c.Tick.NumHours*c.Tick.TicksPerHour)*2 + c.Gap*2
	// for now just do everything
	opt.Channels[0].LedCount = 144
	return ws2811.MakeWS2811(&opt)
}

func main() {
	fmt.Println("starting...")
	c, err := config.ReadConfig(ConfigFile)
	if err != nil {
		fmt.Println("read config error using default:%w", err)
		c = DefaultConfig
	}
	fmt.Printf("~~~~~~~~~~~~~~~\n %+v\n\n", c)

	dev, err := NewLedDisplay(c)
	if err != nil {
		panic(err)
	}

	err = dev.Init()
	if err != nil {
		panic(err)
	}
	defer dev.Fini()

	time.Sleep(1 * time.Second)
	display.Clear(dev)
	time.Sleep(1 * time.Second)

	go startClock(dev, c)

	// // Wait for Shutdown
	fmt.Println("waiting...")
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	<-sigChan

	fmt.Print("shutting down...")

	// try to clear leds when shutting down
	if dev != nil {
		display.Clear(dev)
	}
	time.Sleep(1 * time.Second)
	fmt.Println("done")
}

func startClock(dev *ws2811.WS2811, c *config.Config) {
	if dev == nil {
		return
	}
	for {
		fmt.Println("tick:", time.Now())
		err := display.DisplayTime(time.Now(), c, dev)
		if err != nil {
			fmt.Println("display time error:", err)
			return
		}

		time.Sleep(c.RefreshRate)

		// refresh config, only override if successful. Otherwise don't change config.
		nc, err := config.ReadConfig(ConfigFile)
		if err != nil {
			fmt.Println("read config, no change:%w", err)
		} else {
			c = nc
		}
	}
}
