package main

import (
	"encoding/gob"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/jaredwarren/clock/lib/config"
	"github.com/jaredwarren/clock/lib/display"
	"github.com/jaredwarren/clock/lib/mock"
)

func main() {
	// config, err := readConfig()
	// if err != nil {
	// 	fmt.Println("read config error using default:%w", err)
	// }

	var numLeds = 32
	dev := mock.NewMockDisplay(numLeds)

	err := dev.Init()
	if err != nil {
		panic(err)
	}
	defer dev.Fini()

	// time.Sleep(1 * time.Second)
	clear(dev)
	// time.Sleep(1 * time.Second)

	dev.Leds(0)[0] = getHex("red")
	dev.Leds(0)[1] = 0x9933ff
	dev.Leds(0)[2] = 0xffff80
	dev.Leds(0)[3] = 0x00ffff
	dev.Render()

	t := time.Now()
	// t := time.Now().Add(1 * time.Hour)
	// t := time.Now().Add(3 * time.Minute)
	// t := time.Now().Add(-45 * time.Minute)

	// layout := "2006-01-02T15:04:05.000Z"
	// str := "2014-11-12T11:50:26.371Z"
	// t, _ := time.Parse(layout, str)

	setTime(t, &config.Config{
		Tick: config.TickConfig{
			StartHour:    12, // current hour
			StartLed:     0,
			Reverse:      false,
			TicksPerHour: 4,
			NumHours:     4,
		},
		Num: config.NumConfig{},
	}, dev)

	// go startClock(dev, config)

	// // Wait for Shutdown
	// sigChan := make(chan os.Signal, 1)
	// signal.Notify(sigChan, os.Interrupt)
	// <-sigChan

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
	// time.Sleep(1 * time.Second)
	fmt.Println("done")
}

func startClock(dev display.Displayer, c *config.Config) {
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

func setTime(t time.Time, c *config.Config, dev display.Displayer) {
	//
	h := float64(t.Hour())   // 24h
	m := float64(t.Minute()) // [0, 59]

	tph := float64(c.Tick.TicksPerHour)
	minPerTick := 60 / tph
	mtick := math.Floor(m / minPerTick) // for now just on or off, later I'd like to dim this...
	htick := math.Floor((h - float64(c.Tick.StartHour)) * tph)
	lastLed := mtick + htick

	numTickLeds := c.Tick.NumHours * c.Tick.TicksPerHour
	for i := 0; i < numTickLeds; i++ {
		if i < int(lastLed) {
			dev.Leds(0)[i] = 0x000000 // off
		} else if i > int(lastLed) {
			dev.Leds(0)[i] = getHex("red")
		} else {
			ftick := (minPerTick*mtick - m + minPerTick) / minPerTick
			dev.Leds(0)[i] = rgbToHex(uint8(ftick*255), uint8(0), uint8(0)) // off
		}
	}

	fmt.Println(h, htick, numTickLeds)
	for i := numTickLeds; i < numTickLeds*2; i++ {
		if i < int(htick)+numTickLeds {
			dev.Leds(0)[i] = 0x000000 // off
		} else {
			dev.Leds(0)[i] = 0x008800
		}
	}
	reverseSecondHalf(dev.Leds(0))

	dev.Render()

	// TOOD: turn on or off all leds based on revers and lastLED

}

func reverseSecondHalf(s []uint32) {
	mid := len(s) / 2
	end := len(s) - 1

	for i := mid; i < end; i++ {
		s[i], s[end] = s[end], s[i]
		end--
	}
}

func rgbToHex(r, g, b uint8) uint32 {
	return uint32(r)<<16 | uint32(g)<<8 | uint32(b)
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

func clear(dev display.Displayer) {
	leds := dev.Leds(0)
	for i := 0; i < len(leds); i++ {
		leds[i] = 0x000000
	}
	dev.Render()
}
