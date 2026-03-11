package display

import (
	"testing"
	"time"

	"github.com/jaredwarren/clock/internal/config"
	"github.com/stretchr/testify/assert"
)

// mockDisplayer is a mock implementation of the Displayer interface for testing.
type mockDisplayer struct {
	leds     []uint32
	rendered bool
}

func newMockDisplayer(numLeds int) *mockDisplayer {
	return &mockDisplayer{
		leds: make([]uint32, numLeds),
	}
}

func (m *mockDisplayer) Init() error               { return nil }
func (m *mockDisplayer) Fini()                     {}
func (m *mockDisplayer) Leds(channel int) []uint32 { return m.leds }
func (m *mockDisplayer) Render() error             { m.rendered = true; return nil }

func TestHexRGBConversion(t *testing.T) {
	r, g, b := uint8(0x12), uint8(0x34), uint8(0x56)
	hex := rgbToHex(r, g, b)
	assert.Equal(t, uint32(0x123456), hex)

	r2, g2, b2 := hexToRGB(hex)
	assert.Equal(t, r, r2)
	assert.Equal(t, g, g2)
	assert.Equal(t, b, b2)
}

func TestReversePart(t *testing.T) {
	slice := []uint32{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	reversePart(slice, 2, 7) // reverse from index 2 up to (not including) 7
	expected := []uint32{0, 1, 6, 5, 4, 3, 2, 7, 8, 9}
	assert.Equal(t, expected, slice)
}

func TestApplyBrightness(t *testing.T) {
	leds := []uint32{0xFF0000, 0x00FF00, 0x0000FF}

	// Test with 50% brightness (128/256)
	originalLeds := make([]uint32, len(leds))
	copy(originalLeds, leds)
	applyBrightness(originalLeds, 128)

	r1, g1, b1 := hexToRGB(originalLeds[0])
	assert.InDelta(t, 127, r1, 1)
	assert.Equal(t, uint8(0), g1)
	assert.Equal(t, uint8(0), b1)

	r2, g2, b2 := hexToRGB(originalLeds[1])
	assert.Equal(t, uint8(0), r2)
	assert.InDelta(t, 127, g2, 1)
	assert.Equal(t, uint8(0), b2)

	r3, g3, b3 := hexToRGB(originalLeds[2])
	assert.Equal(t, uint8(0), r3)
	assert.Equal(t, uint8(0), g3)
	assert.InDelta(t, 127, b3, 1)

	// Test full brightness
	fullBrightnessLeds := []uint32{0x808080}
	applyBrightness(fullBrightnessLeds, 256)
	assert.Equal(t, uint32(0x808080), fullBrightnessLeds[0])

	// Test > 255 brightness
	overBrightnessLeds := []uint32{0x808080}
	applyBrightness(overBrightnessLeds, 300)
	assert.Equal(t, uint32(0x808080), overBrightnessLeds[0])

	// Test 0 brightness
	zeroBrightnessLeds := []uint32{0xFFFFFF}
	applyBrightness(zeroBrightnessLeds, 0)
	assert.Equal(t, uint32(0), zeroBrightnessLeds[0])
}

func TestClear(t *testing.T) {
	dev := newMockDisplayer(10)
	for i := range dev.leds {
		dev.leds[i] = 0xFFFFFF
	}

	Clear(dev)

	assert.True(t, dev.rendered)
	for i, led := range dev.leds {
		assert.Equal(t, uint32(0x000000), led, "led %d should be off", i)
	}
}

func TestDisplayTime(t *testing.T) {
	c := &config.Config{
		Brightness: 256, // Use full brightness to simplify color checking
		Tick: config.TickConfig{
			PastColor:    0xff0000,
			PresentColor: 0, // Test fade logic
			FutureColor:  0x00ff00,
			FutureColorB: 0xff00,
			StartHour:    9,
			TicksPerHour: 4,
			NumHours:     4,
		},
		Num: config.NumConfig{
			PastColor:    0xaa0000,
			PresentColor: 0x0000bb,
			FutureColor:  0x00cc00,
		},
		Gap: 2,
	}
	numTickLeds := c.Tick.NumHours * c.Tick.TicksPerHour // 16
	numNumLeds := numTickLeds                            // 16
	totalLeds := numTickLeds + c.Gap + numNumLeds        // 34
	dev := newMockDisplayer(totalLeds)

	// Time: 10:37. h=10, m=37
	// tph=4, minPerTick=15. mtick=floor(37/15)=2.
	// htick=floor((10-9)*4)=4. lastLed=2+4=6.
	testTime := time.Date(2024, 1, 1, 10, 37, 0, 0, time.UTC)

	err := DisplayTime(testTime, c, dev)
	assert.NoError(t, err)
	assert.True(t, dev.rendered)

	t.Run("Tick LEDs", func(t *testing.T) {
		// LEDs 0-5 are past
		for i := 0; i < 6; i++ {
			assert.Equal(t, c.Tick.PastColor, dev.Leds(0)[i], "tick led %d should be past color", i)
		}

		// LED 6 is present (fading)
		r, g, b := hexToRGB(dev.Leds(0)[6])
		assert.InDelta(t, 119, r, 1, "present tick red component")
		assert.Equal(t, uint8(0), g, "present tick green component")
		assert.InDelta(t, 100, b, 1, "present tick blue component")

		assert.Equal(t, c.Tick.FutureColorB, dev.Leds(0)[7], "tick led 7")
		for i := 8; i <= 11; i++ {
			assert.Equal(t, c.Tick.FutureColor, dev.Leds(0)[i], "tick led %d", i)
		}
		for i := 12; i <= 15; i++ {
			assert.Equal(t, c.Tick.FutureColorB, dev.Leds(0)[i], "tick led %d", i)
		}
	})

	t.Run("Number LEDs", func(t *testing.T) {
		for i := 18; i <= 25; i++ {
			assert.Equal(t, c.Num.FutureColor, dev.Leds(0)[i], "num led %d should be future", i)
		}
		for i := 26; i <= 29; i++ {
			assert.Equal(t, c.Num.PresentColor, dev.Leds(0)[i], "num led %d should be present", i)
		}
		for i := 30; i <= 33; i++ {
			assert.Equal(t, c.Num.PastColor, dev.Leds(0)[i], "num led %d should be past", i)
		}
	})

	t.Run("With PresentColor set", func(t *testing.T) {
		c.Tick.PresentColor = 0x123456
		dev := newMockDisplayer(totalLeds)
		err := DisplayTime(testTime, c, dev)
		assert.NoError(t, err)
		assert.Equal(t, c.Tick.PresentColor, dev.Leds(0)[6], "present tick should use PresentColor")
		c.Tick.PresentColor = 0 // reset for other tests
	})
}
