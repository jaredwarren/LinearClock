package display

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/jaredwarren/clock/clock/config"
)

type HTMLDisplay struct {
	leds []uint32
	w    http.ResponseWriter
	c    *config.Config
}

func NewHTMLDisplay(c *config.Config, w http.ResponseWriter) Displayer {
	leds := make([]uint32, c.Tick.NumHours*c.Tick.TicksPerHour*2)
	return &HTMLDisplay{
		leds: leds,
		w:    w,
		c:    c,
	}
}

func (m *HTMLDisplay) Init() error {
	return nil
}

func (m *HTMLDisplay) Fini() {

}

func (m *HTMLDisplay) Leds(channel int) []uint32 {
	return m.leds
}

type Data struct {
	Nums  []Num // TOOD: change to struct, so I can add the hour. map doesn't work because order.
	Ticks []string
}

type Num struct {
	Hour  string
	Color string
}

func (m *HTMLDisplay) Render() error {
	data := &Data{
		Nums:  []Num{},
		Ticks: []string{},
	}

	ticks := m.leds[:len(m.leds)/2]
	for _, led := range ticks {
		r := uint8(led >> 16)
		g := uint8(led >> 8)
		b := uint8(led)

		data.Ticks = append(data.Ticks, fmt.Sprintf("#%02X%02X%02X", r, g, b))
	}
	numbers := m.leds[len(m.leds)/2:]

	// everything is reversed
	j := 0
	for i := len(numbers) - 1; i >= 0; i = i - 4 {
		led := numbers[i]
		r := uint8(led >> 16)
		g := uint8(led >> 8)
		b := uint8(led)

		hour := j + m.c.Tick.StartHour
		if hour > 12 {
			hour = hour - 12
		}

		data.Nums = append(data.Nums, Num{
			Hour:  fmt.Sprintf("%d", hour),
			Color: fmt.Sprintf("#%02X%02X%02X", r, g, b),
		})
		j++
	}

	files := []string{
		"templates/test.html",
		"templates/layout.html",
	}
	tmpl, err := template.New("base").Funcs(template.FuncMap{
		"add": func(i, j int) int {
			return i + j
		},
	}).ParseFiles(files...)
	if err != nil {
		fmt.Fprintf(m.w, "parse template error:%+v", err)
		return err
	}

	err = tmpl.Execute(m.w, data)
	if err != nil {
		fmt.Fprintf(m.w, "exec temp error:%+v", err)
		return err
	}

	return nil
}
