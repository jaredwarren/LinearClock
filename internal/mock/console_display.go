package mock

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/jaredwarren/clock/internal/display"
)

type MockDisplay struct {
	leds []uint32
}

func NewMockDisplay(numLeds int) display.Displayer {
	leds := make([]uint32, numLeds)
	return &MockDisplay{
		leds: leds,
	}
}

func (m *MockDisplay) Init() error {
	return nil
}

func (m *MockDisplay) Fini() {

}

func (m *MockDisplay) Leds(channel int) []uint32 {
	return m.leds
}

func (m *MockDisplay) Render() error {
	for i, led := range m.leds {
		r := int(uint8(led >> 16))
		g := int(uint8(led >> 8))
		b := int(uint8(led))
		color.RGB(r, g, b).Printf("■")

		// temp separater
		if i%4 == 3 {
			fmt.Print("|")
		}
	}
	fmt.Println()
	return nil
}
