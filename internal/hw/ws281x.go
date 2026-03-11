package hw

import (
	"github.com/jaredwarren/clock/internal/config"
	"github.com/jaredwarren/clock/internal/display"
	ws2811 "github.com/rpi-ws281x/rpi-ws281x-go"
)

// ws2811Display adapts *ws2811.WS2811 to display.Displayer.
type ws2811Display struct {
	*ws2811.WS2811
}

func (d *ws2811Display) Init() error {
	return d.WS2811.Init()
}

func (d *ws2811Display) Fini() {
	d.WS2811.Fini()
}

func (d *ws2811Display) Leds(channel int) []uint32 {
	return d.WS2811.Leds(channel)
}

func (d *ws2811Display) Render() error {
	return d.WS2811.Render()
}

// NewLedDisplay creates a WS2811 LED display from config (brightness and LED count).
// Only clockd should import this package; it requires rpi-ws281x C libraries at build time.
func NewLedDisplay(c *config.Config) (display.Displayer, error) {
	opt := ws2811.DefaultOptions
	opt.Channels[0].Brightness = c.Brightness
	numLeds := (c.Tick.NumHours * c.Tick.TicksPerHour) * 2
	opt.Channels[0].LedCount = numLeds
	dev, err := ws2811.MakeWS2811(&opt)
	if err != nil {
		return nil, err
	}
	return &ws2811Display{dev}, nil
}
