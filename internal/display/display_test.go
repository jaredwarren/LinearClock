package display

import (
	"testing"
	"time"

	"github.com/jaredwarren/clock/internal/config"
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
	tests := []struct {
		name string
		r, g, b uint8
		wantHex uint32
	}{
		{"roundtrip 0x12 0x34 0x56", 0x12, 0x34, 0x56, 0x123456},
		{"black", 0, 0, 0, 0},
		{"white", 0xFF, 0xFF, 0xFF, 0xFFFFFF},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hex := rgbToHex(tt.r, tt.g, tt.b)
			if hex != tt.wantHex {
				t.Errorf("rgbToHex(%#x, %#x, %#x) = %#x, want %#x", tt.r, tt.g, tt.b, hex, tt.wantHex)
			}
			r2, g2, b2 := hexToRGB(hex)
			if r2 != tt.r || g2 != tt.g || b2 != tt.b {
				t.Errorf("hexToRGB(%#x) = (%#x, %#x, %#x), want (%#x, %#x, %#x)", hex, r2, g2, b2, tt.r, tt.g, tt.b)
			}
		})
	}
}

func TestReversePart(t *testing.T) {
	tests := []struct {
		name        string
		slice       []uint32
		start, end  int
		want        []uint32
	}{
		{
			name:  "middle segment",
			slice: []uint32{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
			start: 2,
			end:   7,
			want:  []uint32{0, 1, 6, 5, 4, 3, 2, 7, 8, 9},
		},
		{
			name:  "full slice",
			slice: []uint32{1, 2, 3},
			start: 0,
			end:   3,
			want:  []uint32{3, 2, 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reversePart(tt.slice, tt.start, tt.end)
			if len(tt.slice) != len(tt.want) {
				t.Errorf("reversePart(...): len(got) = %d, len(want) = %d", len(tt.slice), len(tt.want))
				return
			}
			for i := range tt.want {
				if tt.slice[i] != tt.want[i] {
					t.Errorf("reversePart(...) at index %d: got %v, want %v", i, tt.slice, tt.want)
					return
				}
			}
		})
	}
}

func TestApplyBrightness(t *testing.T) {
	tests := []struct {
		name       string
		leds       []uint32
		brightness int
		want       []uint32
	}{
		{
			name:       "50 percent brightness",
			leds:       []uint32{0xFF0000, 0x00FF00, 0x0000FF},
			brightness: 128,
			want:       []uint32{0x7F0000, 0x007F00, 0x00007F}, // 255*128/256 ≈ 127
		},
		{
			name:       "full brightness",
			leds:       []uint32{0x808080},
			brightness: 256,
			want:       []uint32{0x808080},
		},
		{
			name:       "over max brightness",
			leds:       []uint32{0x808080},
			brightness: 300,
			want:       []uint32{0x808080},
		},
		{
			name:       "zero brightness",
			leds:       []uint32{0xFFFFFF},
			brightness: 0,
			want:       []uint32{0},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			applyBrightness(tt.leds, tt.brightness)
			if len(tt.leds) != len(tt.want) {
				t.Errorf("applyBrightness(...): len(got) = %d, len(want) = %d", len(tt.leds), len(tt.want))
				return
			}
			for i := range tt.want {
				if got := tt.leds[i]; got != tt.want[i] {
					t.Errorf("applyBrightness(...) at index %d: got %#x, want %#x", i, got, tt.want[i])
					return
				}
			}
		})
	}
}

func TestClear(t *testing.T) {
	tests := []struct {
		name     string
		numLeds  int
		fillWith uint32
	}{
		{"clears 10 leds filled white", 10, 0xFFFFFF},
		{"clears 1 led", 1, 0x123456},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dev := newMockDisplayer(tt.numLeds)
			for i := range dev.leds {
				dev.leds[i] = tt.fillWith
			}
			if err := Clear(dev); err != nil {
				t.Fatalf("Clear() error = %v", err)
			}
			if !dev.rendered {
				t.Error("Clear: expected Render to be called")
			}
			for i, led := range dev.leds {
				if led != 0x000000 {
					t.Errorf("led %d = %#x, want 0x000000", i, led)
				}
			}
		})
	}
}

func TestDisplayTime(t *testing.T) {
	baseCfg := &config.Config{
		Brightness: 256,
		Tick: config.TickConfig{
			PastColor:    0xff0000,
			PresentColor: 0,
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
	numTickLeds := baseCfg.Tick.NumHours * baseCfg.Tick.TicksPerHour
	totalLeds := numTickLeds + baseCfg.Gap + numTickLeds
	testTime := time.Date(2024, 1, 1, 10, 37, 0, 0, time.UTC)

	tests := []struct {
		name   string
		cfg    *config.Config
		check  func(t *testing.T, dev *mockDisplayer, c *config.Config)
	}{
		{
			name: "tick and number LEDs",
			cfg:  baseCfg,
			check: func(t *testing.T, dev *mockDisplayer, c *config.Config) {
				leds := dev.Leds(0)
				for i := 0; i < 6; i++ {
					if leds[i] != c.Tick.PastColor {
						t.Errorf("tick led %d = %#x, want past %#x", i, leds[i], c.Tick.PastColor)
					}
				}
				r, g, b := hexToRGB(leds[6])
				if r < 118 || r > 120 || g != 0 || b < 99 || b > 101 {
					t.Errorf("present tick led 6: r=%d g=%d b=%d, want r≈119 g=0 b≈100", r, g, b)
				}
				if leds[7] != c.Tick.FutureColorB {
					t.Errorf("tick led 7 = %#x, want %#x", leds[7], c.Tick.FutureColorB)
				}
				for i := 8; i <= 11; i++ {
					if leds[i] != c.Tick.FutureColor {
						t.Errorf("tick led %d = %#x, want %#x", i, leds[i], c.Tick.FutureColor)
					}
				}
				for i := 12; i <= 15; i++ {
					if leds[i] != c.Tick.FutureColorB {
						t.Errorf("tick led %d = %#x, want %#x", i, leds[i], c.Tick.FutureColorB)
					}
				}
				for i := 18; i <= 25; i++ {
					if leds[i] != c.Num.FutureColor {
						t.Errorf("num led %d = %#x, want future %#x", i, leds[i], c.Num.FutureColor)
					}
				}
				for i := 26; i <= 29; i++ {
					if leds[i] != c.Num.PresentColor {
						t.Errorf("num led %d = %#x, want present %#x", i, leds[i], c.Num.PresentColor)
					}
				}
				for i := 30; i <= 33; i++ {
					if leds[i] != c.Num.PastColor {
						t.Errorf("num led %d = %#x, want past %#x", i, leds[i], c.Num.PastColor)
					}
				}
			},
		},
		{
			name: "present color set",
			cfg: func() *config.Config {
				cfg := *baseCfg
				cfg.Tick.PresentColor = 0x123456
				return &cfg
			}(),
			check: func(t *testing.T, dev *mockDisplayer, c *config.Config) {
				if got := dev.Leds(0)[6]; got != c.Tick.PresentColor {
					t.Errorf("present tick led 6 = %#x, want PresentColor %#x", got, c.Tick.PresentColor)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dev := newMockDisplayer(totalLeds)
			err := DisplayTime(testTime, tt.cfg, dev)
			if err != nil {
				t.Fatalf("DisplayTime() error = %v", err)
			}
			if !dev.rendered {
				t.Error("DisplayTime: expected Render to be called")
			}
			tt.check(t, dev, tt.cfg)
		})
	}
}

func TestResolveTickColorsForTime(t *testing.T) {
	base := config.TickConfig{
		PastColor:    0x111111,
		PresentColor: 0x222222,
		FutureColor:  0x333333,
		FutureColorB: 0x444444,
	}

	t.Run("no events returns base colors", func(t *testing.T) {
		got := ResolveTickColorsForTime(time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC), base, nil)
		if got.Past != 0x111111 || got.Present != 0x222222 || got.Future != 0x333333 || got.FutureB != 0x444444 {
			t.Errorf("got Past=%#x Present=%#x Future=%#x FutureB=%#x", got.Past, got.Present, got.Future, got.FutureB)
		}
	})

	t.Run("one-time event active on start day applies overrides", func(t *testing.T) {
		events := []config.TickEvent{
			{
				Start:                time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC),
				End:                  time.Date(2024, 6, 15, 14, 0, 0, 0, time.UTC),
				Repeat:               config.RepeatNone,
				PastColorOverride:    0xAAAAAAAA,
				PresentColorOverride: 0,
				FutureColorOverride:  0,
				FutureColorBOverride: 0,
			},
		}
		t.Run("inside window", func(t *testing.T) {
			got := ResolveTickColorsForTime(time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC), base, events)
			if got.Past != 0xAAAAAAAA {
				t.Errorf("Past = %#x, want 0xAAAAAAAA", got.Past)
			}
			if got.Present != 0x222222 {
				t.Errorf("Present unchanged = %#x", got.Present)
			}
		})
		t.Run("outside window same day", func(t *testing.T) {
			got := ResolveTickColorsForTime(time.Date(2024, 6, 15, 15, 0, 0, 0, time.UTC), base, events)
			if got.Past != 0x111111 {
				t.Errorf("Past = %#x, want base 0x111111", got.Past)
			}
		})
		t.Run("different day", func(t *testing.T) {
			got := ResolveTickColorsForTime(time.Date(2024, 6, 16, 12, 0, 0, 0, time.UTC), base, events)
			if got.Past != 0x111111 {
				t.Errorf("Past = %#x, want base 0x111111", got.Past)
			}
		})
	})

	t.Run("daily event matches by time-of-day", func(t *testing.T) {
		events := []config.TickEvent{
			{
				Start:                time.Date(2020, 1, 1, 9, 0, 0, 0, time.UTC),
				End:                  time.Date(2020, 1, 1, 17, 0, 0, 0, time.UTC),
				Repeat:               config.RepeatDaily,
				FutureColorOverride:  0xBBBBBB,
			},
		}
		got := ResolveTickColorsForTime(time.Date(2024, 6, 15, 12, 30, 0, 0, time.UTC), base, events)
		if got.Future != 0xBBBBBB {
			t.Errorf("Future = %#x, want 0xBBBBBB", got.Future)
		}
		if got.Past != 0x111111 {
			t.Errorf("Past unchanged = %#x", got.Past)
		}
	})

	t.Run("later event overrides earlier", func(t *testing.T) {
		events := []config.TickEvent{
			{
				Start:             time.Date(2024, 6, 15, 8, 0, 0, 0, time.UTC),
				End:               time.Date(2024, 6, 15, 18, 0, 0, 0, time.UTC),
				Repeat:            config.RepeatNone,
				PastColorOverride: 0x11111111,
			},
			{
				Start:             time.Date(2024, 6, 15, 8, 0, 0, 0, time.UTC),
				End:               time.Date(2024, 6, 15, 18, 0, 0, 0, time.UTC),
				Repeat:            config.RepeatNone,
				PastColorOverride: 0x22222222,
			},
		}
		got := ResolveTickColorsForTime(time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC), base, events)
		if got.Past != 0x22222222 {
			t.Errorf("Past = %#x, want later event override 0x22222222", got.Past)
		}
	})
}

// TestDisplayTime_eventOverrideOnlyInEventTickBlocks verifies that event color overrides
// apply only to ticks whose time block overlaps the event window (e.g. 10:00-10:29 = two 15-min ticks).
func TestDisplayTime_eventOverrideOnlyInEventTickBlocks(t *testing.T) {
	// StartHour 8, 4 ticks/hour, 6 hours = 24 tick LEDs. Tick 8 = 10:00-10:15, tick 9 = 10:15-10:30.
	cfg := &config.Config{
		Brightness: 256,
		Tick: config.TickConfig{
			PastColor:    0x111111,
			PresentColor: 0x222222,
			FutureColor:  0x333333,
			FutureColorB: 0x444444,
			StartHour:    8,
			TicksPerHour: 4,
			NumHours:     6,
			Events: []config.TickEvent{
				{
					Start:                time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC),
					End:                  time.Date(2024, 6, 15, 10, 29, 0, 0, time.UTC),
					Repeat:               config.RepeatDaily,
					PastColorOverride:    0xEE0000, // red
					PresentColorOverride: 0x00EE00,
					FutureColorOverride:  0x0000EE,
					FutureColorBOverride: 0xEE00EE,
				},
			},
		},
		Num: config.NumConfig{},
		Gap: 2,
	}
	numTickLeds := cfg.Tick.NumHours * cfg.Tick.TicksPerHour
	totalLeds := numTickLeds + cfg.Gap + numTickLeds
	dev := newMockDisplayer(totalLeds)
	t.Run("at 10:15 only ticks 8 and 9 (10:00-10:15 and 10:15-10:30) get event colors", func(t *testing.T) {
		tm := time.Date(2024, 6, 15, 10, 15, 0, 0, time.UTC)
		err := DisplayTime(tm, cfg, dev)
		if err != nil {
			t.Fatalf("DisplayTime: %v", err)
		}
		leds := dev.Leds(0)
		// Tick 8 = past (event override), tick 9 = present (event override). All others = base.
		if leds[8] != 0xEE0000 {
			t.Errorf("tick 8 (10:00-10:15) = %#x, want event past 0xEE0000", leds[8])
		}
		if leds[9] != 0x00EE00 {
			t.Errorf("tick 9 (10:15-10:30) = %#x, want event present 0x00EE00", leds[9])
		}
		if leds[0] != 0x111111 {
			t.Errorf("tick 0 = %#x, want base past 0x111111", leds[0])
		}
		// Tick 10 is future and outside event; should be base FutureColor or FutureColorB (alternating)
		if leds[10] != 0x333333 && leds[10] != 0x444444 {
			t.Errorf("tick 10 (future, outside event) = %#x, want base 0x333333 or 0x444444", leds[10])
		}
		// Tick 7 = 9:45-10:00 (block before event start) must NOT get event colors
		if leds[7] != 0x111111 {
			t.Errorf("tick 7 (9:45-10:00, before event) = %#x, want base past 0x111111", leds[7])
		}
	})
}
