/*

SPI example

*/

package main

import (
	"github.com/stianeikeland/go-rpio"
	"fmt"
)

func main() {
	if err := rpio.Open(); err != nil {
		panic(err)
	}

	if err := rpio.SpiBegin(rpio.Spi0); err != nil {
		panic(err)
	}

	rpio.SpiChipSelect(0) // Select CE0 slave

	
	// Send
	
	rpio.SpiTransmit(0xFF)             // send single byte
 	rpio.SpiTransmit(0xDE, 0xAD, 0xBE) // send several bytes

	data := []byte{'H', 'e', 'l', 'l', 'o', 0}
 	rpio.SpiTransmit(data...)          // send slice of bytes

	
	// Receive

	received := rpio.SpiReceive(5)     // receive 5 bytes, (sends 5 x 0s)
	fmt.Println(received)

	
	// Send & Receive

	buffer := []byte{ 0xDE, 0xED, 0xBE, 0xEF }
	rpio.SpiExchange(buffer)           // buffer is populated with received data
	fmt.Println(buffer)

	rpio.SpiEnd(rpio.Spi0)
	rpio.Close()
}
