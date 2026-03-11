package mock

import (
	"fmt"
	"slices"

	"github.com/fatih/color"
	"github.com/jaredwarren/clock/internal/display"
)

type Format int

const (
	FormatNLine    Format = 0
	FormatLEDLines Format = 1
)

type MockDisplay struct {
	leds   []uint32
	format Format
}

func NewMockDisplay(numLeds int, format Format) display.Displayer {
	leds := make([]uint32, 144)
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
	if m.format == 0 {
		m.renderNLine()
	} else {
		m.renderLEDLines()
	}
	return nil
}

func (m *MockDisplay) renderLEDLines() error {
	ticks := m.leds[:len(m.leds)/2]
	for i, led := range ticks {
		r := int(uint8(led >> 16))
		g := int(uint8(led >> 8))
		b := int(uint8(led))
		color.RGB(r, g, b).Printf("■")

		// temp separater
		if i%4 == 3 {
			fmt.Print("|")
		}
	}
	numbers := m.leds[len(m.leds)/2:]
	//slices.Reverse(numbers)
	for i, led := range numbers {
		// led := m.leds[i]
		r := int(uint8(led >> 16))
		g := int(uint8(led >> 8))
		b := int(uint8(led))
		color.RGB(r, g, b).Printf("•")

		// temp separater
		if i%4 == 3 {
			fmt.Print("|")
		}
	}
	fmt.Println()

	fmt.Println()

	return nil
}

func (m *MockDisplay) renderNLine() error {
	numbers := m.leds[len(m.leds)/2:]
	slices.Reverse(numbers)
	for i, led := range numbers {
		// led := m.leds[i]
		r := int(uint8(led >> 16))
		g := int(uint8(led >> 8))
		b := int(uint8(led))
		color.RGB(r, g, b).Printf("•")

		// temp separater
		if i%4 == 3 {
			fmt.Print("|")
		}
	}
	fmt.Println()

	ticks := m.leds[:len(m.leds)/2]
	for i, led := range ticks {
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

	fmt.Println()

	return nil
}
