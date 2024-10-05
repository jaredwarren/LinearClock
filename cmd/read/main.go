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
	// Open the file for reading
	file, err := os.Open("config.gob")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Create a new decoder
	decoder := gob.NewDecoder(file)

	// Decode the data
	var decodedPeople []Person
	err = decoder.Decode(&decodedPeople)
	if err != nil {
		panic(err)
	}

	// Print the decoded data
	for _, p := range decodedPeople {
		println(p.Name, p.Age)
	}
}
