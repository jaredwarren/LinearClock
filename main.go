package main

import (
	"fmt"
	"time"

	ws2811 "github.com/rpi-ws281x/rpi-ws281x-go"
)

const (
	// brightness = 128
	brightness = 64
	width      = 20
	ledCounts  = 20
)

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {

	types := []int{
		ws2811.WS2811StripRGB,
		ws2811.WS2811StripRBG,
		ws2811.WS2811StripGRB,
		ws2811.WS2811StripGBR,
		ws2811.WS2811StripBRG,
		ws2811.WS2811StripBGR,
	}

	for _, t := range types {
		opt := ws2811.DefaultOptions

		// opt := &ws2811.Option{
		// 	Frequency: ws2811.TargetFreq,
		// 	DmaNum:    ws2811.DefaultDmaNum,
		// 	Channels: []ws2811.ChannelOption{
		// 		{
		// GpioPin:    ws2811.DefaultGpioPin,
		// 			LedCount:   ws2811.DefaultLedCount,
		// 			Brightness: ws2811.DefaultBrightness,
		// 			StripeType: ws2811.WS2812Strip,
		// 			Invert:     false,
		// 			Gamma:      ws2811.,
		// 		},
		// 	},
		// }

		opt.Channels[0].Brightness = brightness
		opt.Channels[0].LedCount = ledCounts
		opt.Channels[0].StripeType = t
		// opt.Channels[0].StripeType = ws2811.WS2811StripGRB
		// opt.Channels[0].StripeType = ws2811.WS2811StripGRB
		// opt.Frequency = 2000
		run(&opt)
		time.Sleep(3 * time.Second)
	}
}

func run(opt *ws2811.Option) {
	fmt.Println("Starting...")

	fmt.Printf("~~~~~~~~~~~~~~~\n %+v\n\n", opt)

	dev, err := ws2811.MakeWS2811(opt)
	checkError(err)

	checkError(dev.Init())
	defer dev.Fini()

	fmt.Printf("~~~~~~~~~~~~~~~\n %+v\n\n", dev)

	for i := 0; i < ledCounts; i++ {
		dev.Leds(0)[i] = 0x000000
	}
	checkError(dev.Render())
	time.Sleep(1 * time.Second)
	fmt.Printf("~~~~~~~~~~~~~~~\n %+v\n\n", dev)
	fmt.Println("off")

	for x := 1; x < 10; x++ {
		dev.Leds(0)[x] = 0xffffff
	}

	for x := 10; x < 20; x++ {
		dev.Leds(0)[x] = 0xff0000
	}

	// for x := 0; x < width; x++ {
	// 	for y := 0; y < height; y++ {
	// 		color := uint32(0xff0000)
	// 		if x > 2 && x < 5 && y > 0 && y < 7 {
	// 			color = 0xffffff
	// 		}
	// 		if x > 0 && x < 7 && y > 2 && y < 5 {
	// 			color = 0xffffff
	// 		}
	// 		dev.Leds(0)[x*height+y] = color
	// 	}
	// }
	checkError(dev.Render())
	fmt.Println("Done!")
}
