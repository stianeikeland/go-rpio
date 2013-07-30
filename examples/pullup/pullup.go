package main

import (
	"fmt"
	"github.com/stianeikeland/go-rpio"
	"os"
)

var (
	// Use mcu pin 22, corresponds to GPIO3 on the pi
	pin = rpio.Pin(22)
)

func main() {
	// Open and map memory to access gpio, check for errors
	if err := rpio.Open(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Unmap gpio memory when done
	defer rpio.Close()

	// Pull up and read value
	pin.PullUp()
	fmt.Printf("PullUp: %d\n", pin.Read())

	// Pull down and read value
	pin.PullDown()
	fmt.Printf("PullDown: %d\n", pin.Read())

}
