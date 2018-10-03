package rpio

import (
	"errors"
)

var (
	SpiMapError = errors.New("SPI registers not mapped correctly - are you root?")
)

const (
	SPI0 = iota // only spi0 supported for now
	SPI1        // aux
	SPI2        // aux
)

const (
	csReg     = 0
	fifoReg   = 1 // TX/RX FIFO
	clkDivReg = 2
)

// Sets SPI pins of given device to SIP mode
// (CE0, CE1, [CE2], SCLK, MOSI, MISO)
// also reset SPI control register
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

// Sets SPI pins of given device to default (Input) mode
func SpiEnd(dev int) {
	var pins = getSpiPins(dev)
	for _, pin := range pins {
		pin.Mode(Input)
	}
}

// Set (maximal) speed [Hz] of SPI clock
// Param speed may be as big as 125MHz in theory, but
// only values up to 31.25MHz are considered relayable.
func SpiSpeed(speed int) {
	const baseFreq = 250 * 1000000
	cdiv := uint32(baseFreq / speed)
	setSpiDiv(cdiv)
}

// Select chip, one of 0, 1, 2
// for selecting slave on CE0, CE1, or CE2
func SpiChipSelect(chip int) { // control & status
	const csMask = 3 // chip select has 2 bits

	cs := uint32(chip & csMask)

	spiMem[csReg] = spiMem[csReg]&^csMask | cs
}

// Transmit all bytes in data to slave
// and simultaneously receives bytes from slave to data
func SpiTransfer(data []byte) { // control & status
	const ta = 1 << 7   // transfer active
	const txd = 1 << 18 // tx fifo can accept data
	const rxd = 1 << 17 // rx fifo contains data
	const done = 1 << 16

	length := len(data)
	i := 0 // data index

	clearSpiTxRxFifo()

	// set TA = 1
	spiMem[csReg] |= ta

	for i < length {
		// Poll TXD writing bytes to SPI_FIFO
		for spiMem[csReg]&txd == 0 {
		}
		spiMem[fifoReg] = uint32(data[i])
		// Poll RXD reading bytes from SPI_FIFO
		for spiMem[csReg]&rxd == 0 {
		}
		data[i] = byte(spiMem[fifoReg])
		i++
	}

	// wait for DONE
	for spiMem[csReg]&done == 0 {
	}

	// Set TA = 0
	spiMem[csReg] &^= ta
}

func setSpiDiv(div uint32) {
	const divMask = 1<<16 - 1 - 1 // cdiv have 16 bits and must be odd (for some reason)
	spiMem[clkDivReg] = div & divMask
}

func clearSpiTxRxFifo() {
	const clearTxRx = 1<<5 | 1<<4

	spiMem[csReg] |= clearTxRx
}

func getSpiPins(dev int) []Pin {
	switch dev {
	case SPI0:
		return []Pin{
			7, 8, 9, 10, 11,
			// 35, 36, 37, 38, 39, // only one set of SPI0 can be set in Spi mode at a time
		}
	case SPI1:
		return []Pin{
			16, 17, 18, 19, 20, 21,
		}
	case SPI2:
		return []Pin{
			40, 41, 42, 43, 44, 45,
		}
	default:
		return []Pin{}
	}
}
