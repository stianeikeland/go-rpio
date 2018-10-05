package rpio

import ()

func Example_SPI() {
	SpiBegin(SPI0) // BCM pins 7 to 11

	SpiSpeed(144000) // 144kHz
	SpiChipSelect(1) // CE1

	SpiTransmit(0xFF)
	SpiTransmit(0xDE, 0xAD)
	SpiTransmit(data...)

	SpiEnd(SPI0)
}
