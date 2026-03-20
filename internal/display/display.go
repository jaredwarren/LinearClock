package display

import (
	"errors"
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
// cfg and d must be non-nil or DisplayTime will return an error.
func DisplayTime(t time.Time, cfg *config.Config, d Displayer) error {
	if cfg == nil {
		return errors.New("display: config is nil")
	}
	if d == nil {
		return errors.New("display: Displayer is nil")
	}

	h := float64(t.Hour())   // 24h format
	m := float64(t.Minute()) // [0, 59]

	// 1. Calculate time components
	ticksPerHour := 4.0 // Default to 4 ticks per hour (15 min)
	ticksPerHourInt := cfg.Tick.TicksPerHour
	if ticksPerHourInt == 0 {
		ticksPerHourInt = 4
	} else {
		ticksPerHour = float64(ticksPerHourInt)
	}
	minPerTick := 60.0 / ticksPerHour
	minuteTick := math.Floor(m / minPerTick)
	// lastLed can be negative when time is before StartHour (all ticks appear "future").
	hourTick := math.Floor((h - float64(cfg.Tick.StartHour)) * ticksPerHour)
	lastLed := minuteTick + hourTick

	leds := d.Leds(0)
	prev := append([]uint32(nil), leds...)
	numTickLeds := cfg.Tick.NumHours * ticksPerHourInt

	// 2. Set tick LEDs (event overrides apply only to ticks whose time block overlaps the event)
	setTickLEDs(leds, numTickLeds, int(lastLed), ticksPerHour, cfg, t)

	// 4. Set "number" LEDs
	setNumberLEDs(leds, numTickLeds, int(hourTick), cfg)

	// 5. Apply global brightness
	applyBrightness(leds, cfg.Brightness)

	// 6. Render to device, with optional transition interpolation.
	if !cfg.Tick.TransitionEnabled || cfg.Tick.TransitionDurationMs <= 0 {
		return d.Render()
	}

	target := append([]uint32(nil), leds...)
	steps := normalizedTransitionSteps(cfg.Tick.TransitionMaxSteps)
	stepDelay := time.Duration(cfg.Tick.TransitionDurationMs) * time.Millisecond / time.Duration(steps)

	// Interpolate from the previous frame to the target frame.
	for step := 1; step <= steps; step++ {
		fraction := float64(step) / float64(steps)
		for i := range leds {
			leds[i] = fade(prev[i], target[i], fraction)
		}
		if err := d.Render(); err != nil {
			return err
		}
		if step < steps && stepDelay > 0 {
			time.Sleep(stepDelay)
		}
	}

	return nil
}

func normalizedTransitionSteps(v int) int {
	if v <= 0 {
		return 1
	}
	if v > 12 { // keep CPU safe on Pi-class devices
		return 12
	}
	return v
}

// effectiveTickColors holds the resolved tick colors (base + event overrides).
type effectiveTickColors struct {
	Past, Present, Future, FutureB uint32
}

// ResolveTickColorsForTime returns tick colors for time t, applying events in order.
// Later events override earlier ones; only non-zero override fields are applied.
// Used for debug output; the display uses per-tick resolution (see resolveColorsForTick).
func ResolveTickColorsForTime(t time.Time, base config.TickConfig, events []config.TickEvent) effectiveTickColors {
	out := effectiveTickColors{
		Past:     base.PastColor,
		Present:  base.PresentColor,
		Future:   base.FutureColor,
		FutureB:  base.FutureColorB,
	}
	for i := range events {
		e := &events[i]
		if !eventMatches(t, e) {
			continue
		}
		if e.PastColorOverride != 0 {
			out.Past = e.PastColorOverride
		}
		if e.PresentColorOverride != 0 {
			out.Present = e.PresentColorOverride
		}
		if e.FutureColorOverride != 0 {
			out.Future = e.FutureColorOverride
		}
		if e.FutureColorBOverride != 0 {
			out.FutureB = e.FutureColorBOverride
		}
	}
	return out
}

// EventMatches reports whether t falls within the event's window.
// Exported for debugging (e.g. /debug/events).
func EventMatches(t time.Time, e *config.TickEvent) bool {
	return eventMatches(t, e)
}

func eventMatches(t time.Time, e *config.TickEvent) bool {
	// Normalize the current time into the event's location so that
	// date comparisons and time-of-day comparisons behave as expected
	// regardless of the local timezone of the running process.
	tt := t.In(e.Start.Location())

	switch e.Repeat {
	case config.RepeatNone:
		// One-time: active only on Start's calendar day (in the event's location),
		// and t in [Start, End].
		if tt.Year() != e.Start.Year() ||
			tt.Month() != e.Start.Month() ||
			tt.Day() != e.Start.Day() {
			return false
		}
		return !tt.Before(e.Start) && !tt.After(e.End)
	case config.RepeatDaily:
		// Daily: compare only time-of-day in the event's location.
		tOD := tt.Hour()*60 + tt.Minute()
		startOD := e.Start.Hour()*60 + e.Start.Minute()
		endOD := e.End.Hour()*60 + e.End.Minute()
		return tOD >= startOD && tOD <= endOD
	default:
		return false
	}
}

// tickTimeRange returns the wall-clock time range [start, end) for tick index i on the same date as t.
// start is inclusive, end is exclusive (tick i covers [start, end)).
func tickTimeRange(i int, t time.Time, tick config.TickConfig) (start, end time.Time) {
	ticksPerHour := tick.TicksPerHour
	if ticksPerHour == 0 {
		ticksPerHour = 4
	}
	minutesPerTick := 60 / ticksPerHour
	loc := t.Location()
	base := time.Date(t.Year(), t.Month(), t.Day(), tick.StartHour, 0, 0, 0, loc)
	start = base.Add(time.Duration(i*minutesPerTick) * time.Minute)
	end = start.Add(time.Duration(minutesPerTick) * time.Minute)
	return start, end
}

// tickOverlapsEvent reports whether the tick's time range [tickStart, tickEnd) overlaps the event's window.
// Uses strict overlap so a tick ending exactly at event start (e.g. 9:45–10:00 vs event 10:00–10:29) does not match.
func tickOverlapsEvent(tickStart, tickEnd time.Time, e *config.TickEvent, dayRef time.Time) bool {
	loc := e.Start.Location()
	tickStart = tickStart.In(loc)
	tickEnd = tickEnd.In(loc)
	// Overlap: tick [start, end) and event [start, end] intersect ↔ tickEnd > eventStart && tickStart < eventEnd
	overlap := func(eventStart, eventEnd time.Time) bool {
		return tickEnd.After(eventStart) && tickStart.Before(eventEnd)
	}
	switch e.Repeat {
	case config.RepeatNone:
		if tickStart.Year() != e.Start.Year() || tickStart.Month() != e.Start.Month() || tickStart.Day() != e.Start.Day() {
			return false
		}
		return overlap(e.Start, e.End)
	case config.RepeatDaily:
		eventStart := time.Date(dayRef.Year(), dayRef.Month(), dayRef.Day(), e.Start.Hour(), e.Start.Minute(), 0, 0, loc)
		eventEnd := time.Date(dayRef.Year(), dayRef.Month(), dayRef.Day(), e.End.Hour(), e.End.Minute(), 0, 0, loc)
		return overlap(eventStart, eventEnd)
	default:
		return false
	}
}

// resolveColorsForTick returns the effective colors for a single tick (for past/present/future role).
// Only events whose time window overlaps this tick's block are applied; later events override earlier.
func resolveColorsForTick(tickIndex, lastLed int, t time.Time, base config.TickConfig, events []config.TickEvent) effectiveTickColors {
	out := effectiveTickColors{
		Past:     base.PastColor,
		Present:  base.PresentColor,
		Future:   base.FutureColor,
		FutureB:  base.FutureColorB,
	}
	tickStart, tickEnd := tickTimeRange(tickIndex, t, base)
	for i := range events {
		e := &events[i]
		if !tickOverlapsEvent(tickStart, tickEnd, e, t) {
			continue
		}
		if e.PastColorOverride != 0 {
			out.Past = e.PastColorOverride
		}
		if e.PresentColorOverride != 0 {
			out.Present = e.PresentColorOverride
		}
		if e.FutureColorOverride != 0 {
			out.Future = e.FutureColorOverride
		}
		if e.FutureColorBOverride != 0 {
			out.FutureB = e.FutureColorBOverride
		}
	}
	return out
}

// setTickLEDs sets the colors for the "tick" portion of the LED strip.
// Event overrides apply only to ticks whose time block overlaps the event window.
func setTickLEDs(leds []uint32, numTickLeds, lastLed int, ticksPerHour float64, cfg *config.Config, t time.Time) {
	ticksPerHourInt := cfg.Tick.TicksPerHour
	if ticksPerHourInt == 0 {
		ticksPerHourInt = 4 // avoid division by zero in modulo
	}
	events := cfg.Tick.Events
	if events == nil {
		events = []config.TickEvent{}
	}

	isAltHour := false

	// Boundary check to prevent panic if configured LEDs exceed available LEDs.
	if numTickLeds > len(leds) {
		numTickLeds = len(leds)
	}

	for i := 0; i < numTickLeds; i++ {
		// Alternate color scheme for each hour block.
		if i%ticksPerHourInt == 0 {
			isAltHour = !isAltHour
		}

		colors := resolveColorsForTick(i, lastLed, t, cfg.Tick, events)

		switch {
		case i < lastLed:
			// Past ticks
			leds[i] = colors.Past
		case i > lastLed:
			// Future ticks
			if isAltHour {
				leds[i] = colors.FutureB
			} else {
				leds[i] = colors.Future
			}
		default: // i == lastLed
			// Current tick
			if colors.Present != 0 {
				leds[i] = colors.Present
			} else {
				// Fade from a "present" color to the past color to show progress through the current tick.
				minPerTick := 60.0 / ticksPerHour
				minute := float64(t.Minute())
				minuteTick := math.Floor(minute / minPerTick)

				// fraction represents the proportion of the current tick remaining.
				fraction := (minPerTick*minuteTick - minute + minPerTick) / minPerTick

				// The fade starts from Num.PresentColor and fades to Tick.PastColor.
				fromColor := colors.Past
				toColor := cfg.Num.PresentColor
				leds[i] = fade(fromColor, toColor, fraction)
			}
		}
	}
}

// setNumberLEDs sets the colors for the "number" or "hour" portion of the LED strip.
func setNumberLEDs(leds []uint32, numTickLeds, hourTick int, cfg *config.Config) {
	ticksPerHour := cfg.Tick.TicksPerHour
	if ticksPerHour == 0 {
		ticksPerHour = 4 // avoid infinite loop in current-hour block
	}

	// "Number" LEDs are assumed to be a second logical group of LEDs,
	// typically after a gap from the tick LEDs.
	start := numTickLeds + cfg.Gap

	// This assumes the number of "hour" LEDs is the same as tick LEDs.
	numHourLEDs := cfg.Tick.NumHours * ticksPerHour

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
			for j := 0; j < ticksPerHour; j++ {
				if idx := ledIndex + j; idx < len(leds) {
					leds[idx] = cfg.Num.PresentColor
				}
			}
			i += ticksPerHour
		}
	}

	// The number LEDs are physically arranged in reverse order.
	reversePart(leds, start, end+1)
}

// fade performs linear interpolation between two colors.
// fraction is from 0.0 to 1.0, controlling the mix; values outside are clamped.
func fade(from, to uint32, fraction float64) uint32 {
	if fraction <= 0 {
		return from
	}
	if fraction >= 1 {
		return to
	}
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

// Clear turns off all LEDs on the display and renders. Returns any error from Render.
func Clear(d Displayer) error {
	if d == nil {
		return errors.New("display: Displayer is nil")
	}
	leds := d.Leds(0)
	for i := range leds {
		leds[i] = 0x000000
	}
	return d.Render()
}
