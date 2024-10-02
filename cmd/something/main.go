package main

import (
	"encoding/gob"
	"os"
)

type Person struct {
	Name string
	Age  int
}

func main() {
	// Create some data to encode
	people := []Person{
		{Name: "Alice", Age: 30},
		{Name: "Bob", Age: 25},
	}

	// Open a file for writing
	file, err := os.Create("people.gob")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Create a new encoder
	encoder := gob.NewEncoder(file)

	// Encode the data
	err = encoder.Encode(people)
	if err != nil {
		panic(err)
	}
}
