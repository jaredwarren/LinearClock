package main

import (
	"encoding/gob"
	"fmt"
	"os"
	"time"

	"github.com/jaredwarren/clock/internal/config"
	ws2811 "github.com/rpi-ws281x/rpi-ws281x-go"
)

const (
	brightness = 64
	ledCount   = 144
)

func main() {

	config, err := readConfig()
	if err != nil {
		panic(err.Error())
	}

	opt := ws2811.DefaultOptions
	opt.Channels[0].Brightness = config.Brightness
	// opt.Channels[0].Brightness = brightness
	opt.Channels[0].LedCount = ledCount

	dev, err := ws2811.MakeWS2811(&opt)
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

	for j := 0; j < 2; j++ {
		i := 1 // skip first

		ticker := time.NewTicker(1 * time.Second)
		done := make(chan bool)
		go func() {
			for {
				select {
				case <-done:
					return
				case <-ticker.C:
					dev.Leds(0)[i] = getHex("red")
					dev.Render()
					i++
				}
			}
		}()

		time.Sleep(60 * time.Second)
		ticker.Stop()
		done <- true
		fmt.Println("Ticker stopped")
	}

	// wipe("red", dev)
	// time.Sleep(10 * time.Second)

	clear(dev)
	time.Sleep(1 * time.Second)
	fmt.Println("done")
}

func readConfig() (*config.Config, error) {
	// Open the file for reading
	file, err := os.Open("people.gob")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Create a new decoder
	decoder := gob.NewDecoder(file)

	// Decode the data
	var config config.Config
	err = decoder.Decode(&config)
	if err != nil {
		panic(err)
	}

	// Print the decoded data
	fmt.Printf("~~~~~~~~~~~~~~~\n %+v\n\n", config)
	return &config, nil
}

func wipe(color string, dev *ws2811.WS2811) {
	for i := 0; i < ledCount; i++ {
		dev.Leds(0)[i] = getHex(color)
		dev.Render()
		time.Sleep(50 * time.Millisecond)
	}
}

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
	for i := 0; i < ledCount; i++ {
		dev.Leds(0)[i] = 0x000000
	}
	dev.Render()
}
