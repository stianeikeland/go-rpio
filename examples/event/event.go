/*

An example of edge event handling by @Drahoslav7, using the go-rpio library

Waits for button to be pressed before exit.

Connect a button between pin 22 and some GND pin.

*/

package main

import (
	"fmt"
	"time"
	"os"
	"github.com/stianeikeland/go-rpio"
)

var (
	// Use mcu pin 22, corresponds to GPIO 3 on the pi
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

	pin.Input()
	pin.PullUp()
	pin.Detect(rpio.FallEdge) // enable falling edge event detection

	fmt.Println("press a button")

	for {
		if pin.EdgeDetected() { // check if event occured
			fmt.Println("button pressed less than a second ago")
			break
		}
		time.Sleep(time.Second)
	}
	pin.Detect(rpio.NoEdge) // disable edge event detection
}
