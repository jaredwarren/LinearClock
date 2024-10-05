package main

import (
	"encoding/gob"
	"fmt"
	"math"
	"os"
	"os/signal"
	"time"

	"github.com/jaredwarren/clock/clock/config"
	ws2811 "github.com/rpi-ws281x/rpi-ws281x-go"
)

func NewLedDisplay(c *config.Config) (*ws2811.WS2811, error) {
	opt := ws2811.DefaultOptions
	opt.Channels[0].Brightness = c.Brightness
	// for now just do tics
	opt.Channels[0].LedCount = c.Tick.TicksPerHour * c.Tick.TicksPerHour
	return ws2811.MakeWS2811(&opt)
}

func main() {
	config, err := readConfig()
	if err != nil {
		fmt.Println("read config error using default:%w", err)
	}

	dev, err := NewLedDisplay(config)
	if err != nil {
		panic(err)
	}

	err = dev.Init()
	if err != nil {
		panic(err)
	}
	defer dev.Fini()

	time.Sleep(1 * time.Second)
	clear(dev)
	time.Sleep(1 * time.Second)

	go startClock(dev, config)

	// Wait for Shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	<-sigChan

	// for j := 0; j < 2; j++ {
	// 	i := 1 // skip first

	// 	ticker := time.NewTicker(1 * time.Second)
	// 	done := make(chan bool)
	// 	go func() {
	// 		for {
	// 			select {
	// 			case <-done:
	// 				return
	// 			case <-ticker.C:
	// 				dev.Leds(0)[i] = getHex("red")
	// 				dev.Render()
	// 				i++
	// 			}
	// 		}
	// 	}()

	// 	time.Sleep(60 * time.Second)
	// 	ticker.Stop()
	// 	done <- true
	// 	fmt.Println("Ticker stopped")
	// }

	// // wipe("red", dev)
	// // time.Sleep(10 * time.Second)

	// try to clear leds when shutting down
	if dev != nil {
		clear(dev)
	}
	time.Sleep(1 * time.Second)
	fmt.Println("done")
}

func startClock(dev *ws2811.WS2811, c *config.Config) {
	if dev == nil {
		return
	}
	for {
		setTime(time.Now(), c, dev)

		time.Sleep(c.RefreshRate)

		// refresh config, only override if successful. Otherwise don't change config.
		nc, err := readConfig()
		if err != nil {
			fmt.Println("read config, no change:%w", err)
		} else {
			c = nc
		}
	}
}

func setTime(t time.Time, c *config.Config, dev *ws2811.WS2811) {
	tph := float64(c.Tick.TicksPerHour)
	//
	m := float64(t.Minute()) // [0, 59]
	minPerTick := 60 / tph
	mtick := math.Floor(m / minPerTick) // for now just on or off

	h := float64(t.Hour())
	htick := h * tph // need to + startHour

	lastLed := mtick + htick

	fmt.Println(lastLed)

	// TOOD: turn on or off all leds based on revers and lastLED

}

var DefaultConfig = &config.Config{
	DisplayMode: "console",
	// General
	Brightness:  64,
	RefreshRate: time.Second * 2,
	// Ticks
	Tick: config.TickConfig{
		StartHour:    18, // 6:00pm
		StartLed:     1,
		TicksPerHour: 4,
		NumHours:     3,
		Reverse:      false,
	},
	// TODO: numbers
}

func readConfig() (*config.Config, error) {
	// Open the file for reading
	file, err := os.Open("config.gob")
	if err != nil {
		return DefaultConfig, err
	}
	defer file.Close()

	// Create a new decoder
	decoder := gob.NewDecoder(file)

	// Decode the data
	var config config.Config
	err = decoder.Decode(&config)
	if err != nil {
		return DefaultConfig, err
	}

	// Print the decoded data
	fmt.Printf("~~~~~~~~~~~~~~~\n %+v\n\n", config)
	return &config, nil
}

// func wipe(color string, dev *ws2811.WS2811) {
// 	for i := 0; i < ledCount; i++ {
// 		dev.Leds(0)[i] = getHex(color)
// 		dev.Render()
// 		time.Sleep(50 * time.Millisecond)
// 	}
// }

func getHex(cs string) uint32 {
	switch cs {
	case "red":
		return 0xff0000
	case "green":
		return 0x00ff00
	case "white":
		return 0xffffff
	case "black":
		return 0x000000
	}
	return 0xffffff
}

func clear(dev *ws2811.WS2811) {
	leds := dev.Leds(0)
	for i := 0; i < len(leds); i++ {
		leds[i] = 0x000000
	}
	dev.Render()
}
