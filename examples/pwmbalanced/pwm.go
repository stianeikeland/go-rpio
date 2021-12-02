/*

A PWM example by @youngkin, using the go-rpio library

Fades a PWM hardware pin in and out using PWM mode balanced (vs. markspace)
*/

package main

import (
	"os"
	"time"

	"github.com/stianeikeland/go-rpio/v4"
)

func main() {
	err := rpio.Open()
	if err != nil {
		os.Exit(1)
	}
	defer rpio.Close()

	pin := rpio.Pin(19)
	pin.Mode(rpio.Pwm)
	pin.Freq(64000)
	pin.DutyCycleWithPwmMode(0, 32, rpio.Balanced)
	// the LED will be blinking at 2000Hz
	// (source frequency divided by cycle length => 64000/32 = 2000)

	// five times smoothly fade in and out
	for i := 0; i < 5; i++ {
		for i := uint32(0); i < 32; i++ { // increasing brightness
			pin.DutyCycleWithPwmMode(i, 32, rpio.Balanced)
			time.Sleep(time.Second / 32)
		}
		for i := uint32(32); i > 0; i-- { // decreasing brightness
			pin.DutyCycleWithPwmMode(i, 32, rpio.Balanced)
			time.Sleep(time.Second / 32)
		}
	}

	pin.DutyCycleWithPwmMode(0, 32, rpio.Balanced)
}
