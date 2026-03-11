package display

import (
	"math"
	"time"

	"github.com/jaredwarren/clock/internal/config"
)

// Displayer represents a device that can display colored LEDs.
type Displayer interface {
	// Init initializes the display device.
	Init() error
	// Fini finalizes the display device, cleaning up resources.
	Fini()
	// Leds returns the slice of LEDs for a given channel, which can be manipulated.
	Leds(channel int) []uint32
	// Render sends the current LED state to the physical device.
	Render() error
}

// DisplayTime calculates and renders the clock face for a given time.
// It orchestrates the drawing of tick marks and hour numbers onto the LED strip.
func DisplayTime(t time.Time, cfg *config.Config, d Displayer) error {
	h := float64(t.Hour())   // 24h format
	m := float64(t.Minute()) // [0, 59]

	// 1. Calculate time components
	ticksPerHour := 4.0 // Default to 4 ticks per hour (15 min)
	if cfg.Tick.TicksPerHour != 0 {
		ticksPerHour = float64(cfg.Tick.TicksPerHour)
	}
	minPerTick := 60.0 / ticksPerHour
	minuteTick := math.Floor(m / minPerTick)
	hourTick := math.Floor((h - float64(cfg.Tick.StartHour)) * ticksPerHour)
	lastLed := minuteTick + hourTick

	leds := d.Leds(0)
	numTickLeds := cfg.Tick.NumHours * cfg.Tick.TicksPerHour

	// 2. Set tick LEDs
	setTickLEDs(leds, numTickLeds, int(lastLed), ticksPerHour, cfg, t)

	// 3. Set "number" LEDs
	setNumberLEDs(leds, numTickLeds, int(hourTick), cfg)

	// TODO: add override here for specific events

	// 4. Apply global brightness
	applyBrightness(leds, cfg.Brightness)

	// 5. Render to device
	return d.Render()
}

// setTickLEDs sets the colors for the "tick" portion of the LED strip.
func setTickLEDs(leds []uint32, numTickLeds, lastLed int, ticksPerHour float64, cfg *config.Config, t time.Time) {
	isAltHour := false

	// Boundary check to prevent panic if configured LEDs exceed available LEDs.
	if numTickLeds > len(leds) {
		numTickLeds = len(leds)
	}

	for i := 0; i < numTickLeds; i++ {
		// Alternate color scheme for each hour block.
		if i%cfg.Tick.TicksPerHour == 0 {
			isAltHour = !isAltHour
		}

		switch {
		case i < lastLed:
			// Past ticks
			leds[i] = cfg.Tick.PastColor
		case i > lastLed:
			// Future ticks
			if isAltHour {
				leds[i] = cfg.Tick.FutureColorB
			} else {
				leds[i] = cfg.Tick.FutureColor
			}
		default: // i == lastLed
			// Current tick
			if cfg.Tick.PresentColor != 0 {
				leds[i] = cfg.Tick.PresentColor
			} else {
				// Fade from a "present" color to the past color to show progress through the current tick.
				minPerTick := 60.0 / ticksPerHour
				minute := float64(t.Minute())
				minuteTick := math.Floor(minute / minPerTick)

				// fraction represents the proportion of the current tick remaining.
				fraction := (minPerTick*minuteTick - minute + minPerTick) / minPerTick

				// The fade starts from Num.PresentColor and fades to Tick.PastColor.
				fromColor := cfg.Tick.PastColor
				toColor := cfg.Num.PresentColor
				leds[i] = fade(fromColor, toColor, fraction)
			}
		}
	}
}

// setNumberLEDs sets the colors for the "number" or "hour" portion of the LED strip.
func setNumberLEDs(leds []uint32, numTickLeds, hourTick int, cfg *config.Config) {
	// "Number" LEDs are assumed to be a second logical group of LEDs,
	// typically after a gap from the tick LEDs.
	start := numTickLeds + cfg.Gap

	// This assumes the number of "hour" LEDs is the same as tick LEDs.
	numHourLEDs := cfg.Tick.NumHours * cfg.Tick.TicksPerHour

	end := start // Used to determine the segment to reverse.

	for i := 0; i < numHourLEDs; {
		ledIndex := start + i
		if ledIndex >= len(leds) {
			break // Boundary check
		}

		switch {
		case i < hourTick:
			leds[ledIndex] = cfg.Num.PastColor
			i++
		case i > hourTick:
			leds[ledIndex] = cfg.Num.FutureColor
			end = ledIndex
			i++
		default: // i == hourTick, the current hour
			// Light up a block of LEDs for the current hour.
			for j := 0; j < cfg.Tick.TicksPerHour; j++ {
				if idx := ledIndex + j; idx < len(leds) {
					leds[idx] = cfg.Num.PresentColor
				}
			}
			i += cfg.Tick.TicksPerHour
		}
	}

	// The number LEDs are physically arranged in reverse order.
	reversePart(leds, start, end+1)
}

// fade performs linear interpolation between two colors.
// fraction is from 0.0 to 1.0, controlling the mix.
func fade(from, to uint32, fraction float64) uint32 {
	r1, g1, b1 := hexToRGB(from)
	r2, g2, b2 := hexToRGB(to)

	r := float64(r1)*(1-fraction) + float64(r2)*fraction
	g := float64(g1)*(1-fraction) + float64(g2)*fraction
	b := float64(b1)*(1-fraction) + float64(b2)*fraction

	return rgbToHex(uint8(r), uint8(g), uint8(b))
}

// applyBrightness scales the brightness of all LEDs.
// brightness is an integer from 0 to 256, where 256 is full brightness.
func applyBrightness(leds []uint32, brightness int) {
	if brightness >= 256 {
		return // Max brightness, no change needed.
	}
	if brightness < 0 {
		brightness = 0
	}

	// Use integer arithmetic for performance. (val * brightness) >> 8 is a fast
	// equivalent of (val * brightness) / 256.
	for i, c := range leds {
		if c == 0 {
			continue
		}
		r, g, b := hexToRGB(c)
		// Use uint16 to prevent overflow during multiplication before shifting.
		r = uint8((uint16(r) * uint16(brightness)) >> 8)
		g = uint8((uint16(g) * uint16(brightness)) >> 8)
		b = uint8((uint16(b) * uint16(brightness)) >> 8)
		leds[i] = rgbToHex(r, g, b)
	}
}

// reversePart reverses a portion of a slice of uint32s in place.
func reversePart(slice []uint32, start, end int) {
	if start < 0 || end > len(slice) || start >= end {
		return // Invalid range
	}
	for i, j := start, end-1; i < j; i, j = i+1, j-1 {
		slice[i], slice[j] = slice[j], slice[i]
	}
}

// rgbToHex converts R, G, B components to a single uint32 color.
func rgbToHex(r, g, b uint8) uint32 {
	return uint32(r)<<16 | uint32(g)<<8 | uint32(b)
}

// hexToRGB converts a uint32 color to its R, G, B components.
func hexToRGB(c uint32) (uint8, uint8, uint8) {
	r := uint8(c >> 16)
	g := uint8(c >> 8)
	b := uint8(c)
	return r, g, b
}

// Clear turns off all LEDs on the display.
func Clear(d Displayer) {
	leds := d.Leds(0)
	for i := range leds {
		leds[i] = 0x000000
	}
	d.Render()
}
