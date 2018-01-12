/*

A PWM example by @Drahoslav7, using the go-rpio library 

Toggles a LED on physical pin 19 (mcu pin 10)
Connect a LED with resistor from pin 19 to ground.

*/

package main

import (
        "os"
        "time"
        "github.com/stianeikeland/go-rpio"
)

func main() {
        err := rpio.Open()
        if err != nil {
                os.Exit(1)
        }
        defer rpio.Close()

        pin := rpio.Pin(19)
        pin.Mode(rpio.Pwm)
        pin.Freq(60000)
        pin.DutyCycle(0, 32)

        for i := 0; i < 5; i++ {
                for i := uint32(0); i < 32; i++ { // increasing brightness
                        pin.DutyCycle(i, 32)
                        time.Sleep(time.Second/32)
                }
                for i := uint32(32); i != 0; i-- { // decreasing brightness
                        pin.DutyCycle(i, 32)
                        time.Sleep(time.Second/32)
                }
        }
}