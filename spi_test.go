package rpio

import ()

func ExampleSpiTransmit() {
	SpiTransmit(0xFF)             // send single byte
	SpiTransmit(0xDE, 0xAD, 0xBE) // send several bytes

	data := []byte{'H', 'e', 'l', 'l', 'o', 0}
	SpiTransmit(data...) // send slice of bytes
}

func ExampleSpiBegin() {
	err := SpiBegin(Spi0) // pins 7 to 11
	if err != nil {
		panic(err)
	}

	// any Spi functions must go there...
	SpiTransmit(42)

	SpiEnd(Spi0)
}
