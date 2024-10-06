package display

import (
	"fmt"
	"math"
	"time"

	"github.com/jaredwarren/clock/clock/config"
)

type Displayer interface {
	Init() error
	Fini()
	Leds(channel int) []uint32
	Render() error
}

func DisplayTime(t time.Time, c *config.Config, dev Displayer) error {
	h := float64(t.Hour())   // 24h
	m := float64(t.Minute()) // [0, 59]

	// calculate tick that matches input time
	tph := float64(4) // default 4, to avoid divide by 0
	if c.Tick.TicksPerHour != 0 {
		tph = float64(c.Tick.TicksPerHour)
	}
	minPerTick := 60 / tph
	mtick := math.Floor(m / minPerTick)
	htick := math.Floor((h - float64(c.Tick.StartHour)) * tph)
	lastLed := mtick + htick

	numTickLeds := c.Tick.NumHours * c.Tick.TicksPerHour
	fmt.Println("numTickLEDS:", numTickLeds, lastLed)
	for i := 0; i < numTickLeds; i++ {
		if i < int(lastLed) {
			// Turn Off tick
			dev.Leds(0)[c.Tick.StartLed+i] = c.Tick.OffColor
		} else if i > int(lastLed) {
			// Turn On tick
			dev.Leds(0)[c.Tick.StartLed+i] = c.Tick.OnColor
		} else {
			// fade linearly between off and on color
			ftick := (minPerTick*mtick - m + minPerTick) / minPerTick

			ru8, gu8, bu8 := hexToRGB(c.Tick.OnColor)
			r := ftick * float64(ru8)
			g := ftick * float64(gu8)
			b := ftick * float64(bu8)

			oru8, ogu8, obu8 := hexToRGB(c.Tick.OffColor)
			or := (1 - ftick) * float64(oru8)
			og := (1 - ftick) * float64(ogu8)
			ob := (1 - ftick) * float64(obu8)

			dev.Leds(0)[c.Tick.StartLed+i] = rgbToHex(uint8(r+or), uint8(g+og), uint8(b+ob))
		}
	}

	// Set "number" leds.
	// assume: same number of leds as ticks,
	// assume: numbers follow ticks
	// assume: reverse order
	for i := numTickLeds; i < numTickLeds*2; i++ {
		if i < int(htick)+numTickLeds {
			dev.Leds(0)[i+c.Gap] = c.Num.PastColor
		} else if i > int(htick)+numTickLeds {
			dev.Leds(0)[i+c.Gap] = c.Num.FutureColor
		} else {
			for j := i; j < i+c.Tick.TicksPerHour; j++ {
				dev.Leds(0)[j+c.Gap] = c.Num.CurrentColor
			}
			i = i + c.Tick.TicksPerHour - 1
		}
	}
	reverseSecondHalf(dev.Leds(0))

	fmt.Print("rendering...")
	defer func() {
		fmt.Println("done")
	}()
	return dev.Render()
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

func hexToRGB(c uint32) (uint8, uint8, uint8) {
	r := (uint8(c >> 16))
	g := (uint8(c >> 8))
	b := (uint8(c))
	return r, g, b
}

func Clear(dev Displayer) {
	leds := dev.Leds(0)
	for i := 0; i < len(leds); i++ {
		leds[i] = 0x000000
	}
	dev.Render()
}