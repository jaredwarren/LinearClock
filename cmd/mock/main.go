package main

import (
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/jaredwarren/clock/lib/config"
	"github.com/jaredwarren/clock/lib/display"
	"github.com/jaredwarren/clock/lib/mock"
)

var DefaultConfig = &config.Config{
	RefreshRate: 1 * time.Minute,
	Tick: config.TickConfig{
		OnColor:      0x00ff00,
		OffColor:     0xFF0000,
		StartHour:    17, // for testing set to curren thour or lower
		StartLed:     0,
		Reverse:      false,
		TicksPerHour: 4,
		NumHours:     4,
	},
	Num: config.NumConfig{
		PastColor:    0xff0000,
		FutureColor:  0x00ff00,
		CurrentColor: 0x0000ff,
	},
	Gap: 4, // half
}

func main() {
	c, err := config.ReadConfig("config.gob")
	if err != nil {
		fmt.Println("read config error using default:%w", err)
		c = DefaultConfig
	}

	var numLeds = (c.Tick.NumHours*c.Tick.TicksPerHour)*2 + c.Gap*2
	dev := mock.NewMockDisplay(numLeds)

	err = dev.Init()
	if err != nil {
		panic(err)
	}
	defer dev.Fini()

	// time.Sleep(1 * time.Second)
	display.Clear(dev)
	time.Sleep(1 * time.Second)

	// dev.Leds(0)[0] = getHex("red")
	// dev.Leds(0)[1] = 0x9933ff
	// dev.Leds(0)[2] = 0xffff80
	// dev.Leds(0)[3] = 0x00ffff
	// dev.Render()

	// t := time.Now()
	// t := time.Now().Add(5 * time.Hour)
	// t := time.Now().Add(25 * time.Minute)
	// t := time.Now().Add(-45 * time.Minute)

	// layout := "2006-01-02T15:04:05.000Z"
	// str := "2014-11-12T11:59:26.371Z"
	// str := "2014-11-12T12:01:26.371Z"
	// t, _ := time.Parse(layout, str)

	// setTime(t, &config.Config{
	// 	Tick: config.TickConfig{
	// 		OnColor:      0x00ff00,
	// 		OffColor:     0xFF0000,
	// 		StartHour:    10, // for testing set to curren thour or lower
	// 		StartLed:     0,
	// 		Reverse:      false,
	// 		TicksPerHour: 4,
	// 		NumHours:     4,
	// 	},
	// 	Num: config.NumConfig{
	// 		PastColor:    0xff0000,
	// 		FutureColor:  0x00ff00,
	// 		CurrentColor: 0x0000ff,
	// 	},
	// }, dev)

	// setTime(t, &config.Config{
	// 	Tick: config.TickConfig{
	// 		OnColor:      0x00ff00,
	// 		OffColor:     0x000000,
	// 		StartHour:    10, // for testing set to curren thour or lower
	// 		StartLed:     0,
	// 		Reverse:      false,
	// 		TicksPerHour: 4,
	// 		NumHours:     4,
	// 	},
	// 	Num: config.NumConfig{
	// 		PastColor:    0x000000,
	// 		FutureColor:  0x00ff00,
	// 		CurrentColor: 0x00ff00,
	// 	},
	// }, dev)

	// setTime(t, &config.Config{
	// 	Tick: config.TickConfig{
	// 		OnColor:      0x000000,
	// 		OffColor:     0x00ff00,
	// 		StartHour:    10, // for testing set to curren thour or lower
	// 		StartLed:     0,
	// 		Reverse:      false,
	// 		TicksPerHour: 4,
	// 		NumHours:     4,
	// 	},
	// 	Num: config.NumConfig{
	// 		PastColor:    0x00ff00,
	// 		FutureColor:  0x000000,
	// 		CurrentColor: 0xff0000,
	// 	},
	// }, dev)

	go startClock(dev, c)

	// // Wait for Shutdown
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
		display.Clear(dev)
	}
	// time.Sleep(1 * time.Second)
	fmt.Println("done")
}

func startClock(dev display.Displayer, c *config.Config) {
	if dev == nil {
		return
	}
	for {
		display.DisplayTime(time.Now(), c, dev)

		if c.RefreshRate > 0 {
			time.Sleep(c.RefreshRate)
		} else {
			time.Sleep(2 * time.Second)
		}

		// refresh config, only override if successful. Otherwise don't change config.
		nc, err := config.ReadConfig("config.gob")
		if err != nil {
			fmt.Println("read config, no change:%w", err)
		} else {
			c = nc
		}
	}
}
