// SPI functionality is implemented here
package rpio

import (
	"errors"
)

const (
	SPI0 = iota // only SPI0 supported for now
	SPI1        // aux
	SPI2        // aux
)

const (
	csReg     = 0
	fifoReg   = 1 // TX/RX FIFO
	clkDivReg = 2
)

var (
	SpiMapError = errors.New("SPI registers not mapped correctly - are you root?")
)

// Sets SPI pins of given device to SPI mode
// (CE0, CE1, [CE2], SCLK, MOSI, MISO).
// It also resets SPI control register.
func SpiBegin(dev int) error {
	spiMem[csReg] = 0 // reset spi settings to default
	if spiMem[csReg] == 0 {
		// this should not read only zeroes after reset -> mem map failed
		return SpiMapError
	}

	for _, pin := range getSpiPins(dev) {
		pin.Mode(Spi)
	}

	clearSpiTxRxFifo()
	setSpiDiv(128)
	return nil
}

// Sets SPI pins of given device to default (Input) mode.
func SpiEnd(dev int) {
	var pins = getSpiPins(dev)
	for _, pin := range pins {
		pin.Mode(Input)
	}
}

// Set (maximal) speed [Hz] of SPI clock.
// Param speed may be as big as 125MHz in theory, but
// only values up to 31.25MHz are considered relayable.
func SpiSpeed(speed int) {
	const baseFreq = 250 * 1000000
	cdiv := uint32(baseFreq / speed)
	setSpiDiv(cdiv)
}

// Select chip, one of 0, 1, 2
// for selecting slave on CE0, CE1, or CE2 pin
func SpiChipSelect(chip int) {
	const csMask = 3 // chip select has 2 bits

	cs := uint32(chip & csMask)

	spiMem[csReg] = spiMem[csReg]&^csMask | cs
}

// SpiTransmit takes one or more bytes and send them to slave.
//
// Data received from slave are ignored.
// Use spread operator to send slice of bytes.
func SpiTransmit(data ...byte) {
	SpiExchange(append(data[:0:0], data...)) // clone data because it will be rewriten by received bytes
}

// SpiReceive receives n bytes from slave.
//
// Note that n zeroed bytes are send to slave as side effect.
func SpiReceive(n int) []byte {
	data := make([]byte, n, n)
	SpiExchange(data)
	return data
}

// Transmit all bytes in data to slave
// and simultaneously receives bytes from slave to data.
//
// If you want to only send or only receive, use SpiTransmit/SpiReceive
func SpiExchange(data []byte) {
	const ta = 1 << 7   // transfer active
	const txd = 1 << 18 // tx fifo can accept data
	const rxd = 1 << 17 // rx fifo contains data
	const done = 1 << 16

	clearSpiTxRxFifo()

	// set TA = 1
	spiMem[csReg] |= ta

	for i := range data {
		// wait for TXD
		for spiMem[csReg]&txd == 0 {
		}
		// write bytes to SPI_FIFO
		spiMem[fifoReg] = uint32(data[i])

		// wait for RXD
		for spiMem[csReg]&rxd == 0 {
		}
		// read bytes from SPI_FIFO
		data[i] = byte(spiMem[fifoReg])
	}

	// wait for DONE
	for spiMem[csReg]&done == 0 {
	}

	// Set TA = 0
	spiMem[csReg] &^= ta
}

// set spi clock divider value
func setSpiDiv(cdiv uint32) {
	const cdivMask = 1<<16 - 1 - 1 // cdiv have 16 bits and must be odd (for some reason)
	spiMem[clkDivReg] = div & divMask
}

// clear both FIFOs
func clearSpiTxRxFifo() {
	const clearTxRx = 1<<5 | 1<<4
	spiMem[csReg] |= clearTxRx
}

func getSpiPins(dev int) []Pin {
	switch dev {
	case SPI0:
		return []Pin{7, 8, 9, 10, 11}
		// ommit 35, 36, 37, 38, 39 - only one set of SPI0 can be set in Spi mode at a time
	case SPI1:
		return []Pin{16, 17, 18, 19, 20, 21}
	case SPI2:
		return []Pin{40, 41, 42, 43, 44, 45}
	default:
		return []Pin{}
	}
}
