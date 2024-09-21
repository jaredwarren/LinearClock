package main

import "fmt"

// "github.com/jgarff/rpi_ws281x/golang/ws2811"

const (
	brightness = 128
	width      = 8
	height     = 8
	ledCounts  = width * height
)

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	fmt.Println(".......")
}
