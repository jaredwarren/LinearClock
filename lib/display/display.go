package display

type Displayer interface {
	Init() error
	Fini()
	Leds(channel int) []uint32
	Render() error
}
