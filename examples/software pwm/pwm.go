/*

A Software Based PWM example by @Ronin11, using the go-rpio library

Toggles a LED on physical pin 10
Connect a LED with resistor from pin 10 to ground.

*/

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/Ronin11/go-rpio"
)

const pin = rpio.Pin(10)

func main() {

	if err := rpio.Open(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Unmap gpio memory when done
	defer rpio.Close()

	//Creates the PWM Signal running on the pin, at 2KHz, with a 0% PWM cycle.
	pwm := rpio.CreateSofwarePWM(pin, 2000, 0, 32)
	
	pwm.Start()
	defer pwm.Stop()
	// five times smoothly fade in and out
	for i := 0; i < 5; i++ {
		for i := uint32(0); i < 32; i++ { // increasing brightness
				pwm.SetDutyCycle(i, 32)
				time.Sleep(time.Second/32)
		}
		for i := uint32(32); i > 0; i-- { // decreasing brightness
				pwm.SetDutyCycle(i, 32)
				time.Sleep(time.Second/32)
		}
	}
}