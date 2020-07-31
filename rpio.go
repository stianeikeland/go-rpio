/*
Package rpio provides GPIO access on the Raspberry PI without any need
for external c libraries (eg. WiringPi or BCM2835).

Supports simple operations such as:
	- Pin mode/direction (input/output/clock/pwm,alt0,alt1,alt2,alt3,alt4,alt5)
	- Pin write (high/low)
	- Pin read (high/low)
	- Pin edge detection (no/rise/fall/any)
	- Pull up/down/off
Also clock/pwm related oparations:
	- Set Clock frequency
	- Set Duty cycle
And SPI oparations:
	- SPI transmit/recieve/exchange bytes
	- Chip select
	- Set speed

Example of use:

	rpio.Open()
	defer rpio.Close()

	pin := rpio.Pin(4)
	pin.Output()

	for {
		pin.Toggle()
		time.Sleep(time.Second)
	}

The library use the raw BCM2835 pinouts, not the ports as they are mapped
on the output pins for the raspberry pi, and not the wiringPi convention.

            Rev 2 and 3 Raspberry Pi                        Rev 1 Raspberry Pi (legacy)
  +-----+---------+----------+---------+-----+      +-----+--------+----------+--------+-----+
  | BCM |   Name  | Physical | Name    | BCM |      | BCM | Name   | Physical | Name   | BCM |
  +-----+---------+----++----+---------+-----+      +-----+--------+----++----+--------+-----+
  |     |    3.3v |  1 || 2  | 5v      |     |      |     | 3.3v   |  1 ||  2 | 5v     |     |
  |   2 |   SDA 1 |  3 || 4  | 5v      |     |      |   0 | SDA    |  3 ||  4 | 5v     |     |
  |   3 |   SCL 1 |  5 || 6  | 0v      |     |      |   1 | SCL    |  5 ||  6 | 0v     |     |
  |   4 | GPIO  7 |  7 || 8  | TxD     | 14  |      |   4 | GPIO 7 |  7 ||  8 | TxD    |  14 |
  |     |      0v |  9 || 10 | RxD     | 15  |      |     | 0v     |  9 || 10 | RxD    |  15 |
  |  17 | GPIO  0 | 11 || 12 | GPIO  1 | 18  |      |  17 | GPIO 0 | 11 || 12 | GPIO 1 |  18 |
  |  27 | GPIO  2 | 13 || 14 | 0v      |     |      |  21 | GPIO 2 | 13 || 14 | 0v     |     |
  |  22 | GPIO  3 | 15 || 16 | GPIO  4 | 23  |      |  22 | GPIO 3 | 15 || 16 | GPIO 4 |  23 |
  |     |    3.3v | 17 || 18 | GPIO  5 | 24  |      |     | 3.3v   | 17 || 18 | GPIO 5 |  24 |
  |  10 |    MOSI | 19 || 20 | 0v      |     |      |  10 | MOSI   | 19 || 20 | 0v     |     |
  |   9 |    MISO | 21 || 22 | GPIO  6 | 25  |      |   9 | MISO   | 21 || 22 | GPIO 6 |  25 |
  |  11 |    SCLK | 23 || 24 | CE0     | 8   |      |  11 | SCLK   | 23 || 24 | CE0    |   8 |
  |     |      0v | 25 || 26 | CE1     | 7   |      |     | 0v     | 25 || 26 | CE1    |   7 |
  |   0 |   SDA 0 | 27 || 28 | SCL 0   | 1   |      +-----+--------+----++----+--------+-----+
  |   5 | GPIO 21 | 29 || 30 | 0v      |     |
  |   6 | GPIO 22 | 31 || 32 | GPIO 26 | 12  |
  |  13 | GPIO 23 | 33 || 34 | 0v      |     |
  |  19 | GPIO 24 | 35 || 36 | GPIO 27 | 16  |
  |  26 | GPIO 25 | 37 || 38 | GPIO 28 | 20  |
  |     |      0v | 39 || 40 | GPIO 29 | 21  |
  +-----+---------+----++----+---------+-----+

See the spec for full details of the BCM2835 controller:

https://www.raspberrypi.org/documentation/hardware/raspberrypi/bcm2835/BCM2835-ARM-Peripherals.pdf
and https://elinux.org/BCM2835_datasheet_errata - for errors in that spec

Changes to support the BCM2711, used on the Raspberry Pi 4, were cribbed from https://github.com/RPi-Distro/raspi-gpio/

*/
package rpio

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"reflect"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

type Mode uint8
type Pin uint8
type State uint8
type Pull uint8
type Edge uint8

// Memory offsets for gpio, see the spec for more details
const (
	bcm2835Base = 0x20000000
	gpioOffset  = 0x200000
	clkOffset   = 0x101000
	pwmOffset   = 0x20C000
	spiOffset   = 0x204000
	intrOffset  = 0x00B000

	memLength = 4096
)

// BCM 2711 has a different mechanism for pull-up/pull-down/enable
const (
	GPPUPPDN0 = 57 // Pin pull-up/down for pins 15:0
	GPPUPPDN1 = 58 // Pin pull-up/down for pins 31:16
	GPPUPPDN2 = 59 // Pin pull-up/down for pins 47:32
	GPPUPPDN3 = 60 // Pin pull-up/down for pins 57:48
)

var (
	gpioBase int64
	clkBase  int64
	pwmBase  int64
	spiBase  int64
	intrBase int64

	irqsBackup uint64
)

func init() {
	base := getBase()
	gpioBase = base + gpioOffset
	clkBase = base + clkOffset
	pwmBase = base + pwmOffset
	spiBase = base + spiOffset
	intrBase = base + intrOffset
}

// Pin mode, a pin can be set in Input or Output, Clock or Pwm mode
const (
	Input Mode = iota
	Output
	Clock
	Pwm
	Spi
	Alt0
	Alt1
	Alt2
	Alt3
	Alt4
	Alt5
)

// State of pin, High / Low
const (
	Low State = iota
	High
)

// Pull Up / Down / Off
const (
	PullOff Pull = iota
	PullDown
	PullUp
	PullNone
)

// Edge events
const (
	NoEdge Edge = iota
	RiseEdge
	FallEdge
	AnyEdge = RiseEdge | FallEdge
)

// Arrays for 8 / 32 bit access to memory and a semaphore for write locking
var (
	memlock  sync.Mutex
	gpioMem  []uint32
	clkMem   []uint32
	pwmMem   []uint32
	spiMem   []uint32
	intrMem  []uint32
	gpioMem8 []uint8
	clkMem8  []uint8
	pwmMem8  []uint8
	spiMem8  []uint8
	intrMem8 []uint8
)

// Input: Set pin as Input
func (pin Pin) Input() {
	PinMode(pin, Input)
}

// Output: Set pin as Output
func (pin Pin) Output() {
	PinMode(pin, Output)
}

// Clock: Set pin as Clock
func (pin Pin) Clock() {
	PinMode(pin, Clock)
}

// Pwm: Set pin as Pwm
func (pin Pin) Pwm() {
	PinMode(pin, Pwm)
}

// High: Set pin High
func (pin Pin) High() {
	WritePin(pin, High)
}

// Low: Set pin Low
func (pin Pin) Low() {
	WritePin(pin, Low)
}

// Toggle pin state
func (pin Pin) Toggle() {
	TogglePin(pin)
}

// Freq: Set frequency of Clock or Pwm pin (see doc of SetFreq)
func (pin Pin) Freq(freq int) {
	SetFreq(pin, freq)
}

// DutyCycle: Set duty cycle for Pwm pin (see doc of SetDutyCycle)
func (pin Pin) DutyCycle(dutyLen, cycleLen uint32) {
	SetDutyCycle(pin, dutyLen, cycleLen)
}

// Mode: Set pin Mode
func (pin Pin) Mode(mode Mode) {
	PinMode(pin, mode)
}

// Write: Set pin state (high/low)
func (pin Pin) Write(state State) {
	WritePin(pin, state)
}

// Read pin state (high/low)
func (pin Pin) Read() State {
	return ReadPin(pin)
}

// Pull: Set a given pull up/down mode
func (pin Pin) Pull(pull Pull) {
	PullMode(pin, pull)
}

// PullUp: Pull up pin
func (pin Pin) PullUp() {
	PullMode(pin, PullUp)
}

// PullDown: Pull down pin
func (pin Pin) PullDown() {
	PullMode(pin, PullDown)
}

// PullOff: Disable pullup/down on pin
func (pin Pin) PullOff() {
	PullMode(pin, PullOff)
}

func (pin Pin) ReadPull() Pull {
	if !isBCM2711() {
		return PullNone // Can't read pull-up/pull-down state on other Pi boards
	}

	reg := GPPUPPDN0 + (uint8(pin) >> 4)
	bits := gpioMem[reg] >> ((uint8(pin) & 0xf) << 1) & 0x3
	switch bits {
	case 0:
		return PullOff
	case 1:
		return PullUp
	case 2:
		return PullDown
	default:
		return PullNone // Invalid
	}
}

// Detect: Enable edge event detection on pin
func (pin Pin) Detect(edge Edge) {
	DetectEdge(pin, edge)
}

// EdgeDetected checks edge event on pin
func (pin Pin) EdgeDetected() bool {
	return EdgeDetected(pin)
}

// Get gpio mode
func (pin Pin) ReadMode() uint32 {
	return ReadPinMode(pin)
}

// PinMode sets the mode of a given pin (Input, Output, Clock, Pwm or Spi)
//
// Clock is possible only for pins 4, 5, 6, 20, 21.
// Pwm is possible only for pins 12, 13, 18, 19.
//
// Spi mode should not be set by this directly, use SpiBegin instead.
func PinMode(pin Pin, mode Mode) {

	// Pin fsel register, 0 or 1 depending on bank
	fselReg := uint8(pin) / 10
	shift := (uint8(pin) % 10) * 3
	f := uint32(0)

	const in = 0   // 000
	const out = 1  // 001
	const alt0 = 4 // 100
	const alt1 = 5 // 101
	const alt2 = 6 // 110
	const alt3 = 7 // 111
	const alt4 = 3 // 011
	const alt5 = 2 // 010

	switch mode {
	case Input:
		f = in
	case Output:
		f = out
	case Clock:
		switch pin {
		case 4, 5, 6, 32, 34, 42, 43, 44:
			f = alt0
		case 20, 21:
			f = alt5
		default:
			return
		}
	case Pwm:
		switch pin {
		case 12, 13, 40, 41, 45:
			f = alt0
		case 18, 19:
			f = alt5
		default:
			return
		}
	case Spi:
		switch pin {
		case 7, 8, 9, 10, 11: // SPI0
			f = alt0
		case 35, 36, 37, 38, 39: // SPI0
			f = alt0
		case 16, 17, 18, 19, 20, 21: // SPI1
			f = alt4
		case 40, 41, 42, 43, 44, 45: // SPI2
			f = alt4
		default:
			return
		}
	case Alt0:
		f = alt0
	case Alt1:
		f = alt1
	case Alt2:
		f = alt2
	case Alt3:
		f = alt3
	case Alt4:
		f = alt4
	case Alt5:
		f = alt5
	}

	memlock.Lock()
	defer memlock.Unlock()

	const pinMask = 7 // 111 - pinmode is 3 bits

	gpioMem[fselReg] = (gpioMem[fselReg] &^ (pinMask << shift)) | (f << shift)
}

// Read pin Mod status。（gpio readall）
// const in = 0   // 000
// const out = 1  // 001
// const alt0 = 4 // 100
// const alt1 = 5 // 101
// const alt2 = 6 // 110
// const alt3 = 7 // 111
// const alt4 = 3 // 011
// const alt5 = 2 // 010
func ReadPinMode(pin Pin) uint32 {
	fselReg := uint8(pin) / 10
	shift := (uint8(pin) % 10) * 3
	return gpioMem[fselReg] >> shift & 7
}

// WritePin sets a given pin High or Low
// by setting the clear or set registers respectively
func WritePin(pin Pin, state State) {
	p := uint8(pin)

	// Set register, 7 / 8 depending on bank
	// Clear register, 10 / 11 depending on bank
	setReg := p/32 + 7
	clearReg := p/32 + 10

	memlock.Lock()

	if state == Low {
		gpioMem[clearReg] = 1 << (p & 31)
	} else {
		gpioMem[setReg] = 1 << (p & 31)
	}
	memlock.Unlock() // not deferring saves ~600ns
}

// ReadPin reads the state of a pin
func ReadPin(pin Pin) State {
	// Input level register offset (13 / 14 depending on bank)
	levelReg := uint8(pin)/32 + 13

	if (gpioMem[levelReg] & (1 << uint8(pin&31))) != 0 {
		return High
	}

	return Low
}

// TogglePin: Toggle a pin state (high -> low -> high)
func TogglePin(pin Pin) {
	p := uint8(pin)

	setReg := p/32 + 7
	clearReg := p/32 + 10
	levelReg := p/32 + 13

	bit := uint32(1 << (p & 31))

	memlock.Lock()

	if (gpioMem[levelReg] & bit) != 0 {
		gpioMem[clearReg] = bit
	} else {
		gpioMem[setReg] = bit
	}
	memlock.Unlock()
}

// DetectEdge: Enable edge event detection on pin.
//
// Combine with pin.EdgeDetected() to check whether event occured.
//
// Note that using this function might conflict with the same functionality of other gpio library.
//
// It also clears previously detected event of this pin if there was any.
//
// Note that call with RiseEdge will disable previously set FallEdge detection and vice versa.
// You have to call with AnyEdge, to enable detection for both edges.
// To disable previously enabled detection call it with NoEdge.
//
// WARNING: this might make your Pi unresponsive, if this happens, you should either run the code as root,
// or add `dtoverlay=gpio-no-irq` to `/boot/config.txt` and restart your pi,
func DetectEdge(pin Pin, edge Edge) {
	if edge != NoEdge {
		// disable GPIO event interruption to prevent freezing in some cases
		DisableIRQs(1<<49 | 1<<52) // gpio_int[0] and gpio_int[3]
	}

	p := uint8(pin)

	// Rising edge detect enable register (19/20 depending on bank)
	// Falling edge detect enable register (22/23 depending on bank)
	// Event detect status register (16/17)
	renReg := p/32 + 19
	fenReg := p/32 + 22
	edsReg := p/32 + 16

	bit := uint32(1 << (p & 31))

	if edge&RiseEdge > 0 { // set bit
		gpioMem[renReg] |= bit
	} else { // clear bit
		gpioMem[renReg] &^= bit
	}
	if edge&FallEdge > 0 { // set bit
		gpioMem[fenReg] |= bit
	} else { // clear bit
		gpioMem[fenReg] &^= bit
	}

	gpioMem[edsReg] = bit // to clear outdated detection
}

// EdgeDetected checks whether edge event occured since last call
// or since detection was enabled
//
// There is no way (yet) to handle interruption caused by edge event, you have to use polling.
//
// Event detection has to be enabled first, by pin.Detect(edge)
func EdgeDetected(pin Pin) bool {
	p := uint8(pin)

	// Event detect status register (16/17)
	edsReg := p/32 + 16

	test := gpioMem[edsReg] & (1 << (p & 31))
	gpioMem[edsReg] = test // set bit to clear it
	return test != 0
}

func PullMode(pin Pin, pull Pull) {

	memlock.Lock()
	defer memlock.Unlock()

	if isBCM2711() {
		pullreg := GPPUPPDN0 + (pin >> 4)
		pullshift := (pin & 0xf) << 1

		var p uint32

		switch pull {
		case PullOff:
			p = 0
		case PullUp:
			p = 1
		case PullDown:
			p = 2
		}

		// This is verbatim C code from raspi-gpio.c
		pullbits := gpioMem[pullreg]
		pullbits &= ^(3 << pullshift)
		pullbits |= (p << pullshift)
		gpioMem[pullreg] = pullbits
	} else {
		// Pull up/down/off register has offset 38 / 39, pull is 37
		pullClkReg := pin/32 + 38
		pullReg := 37
		shift := pin % 32

		switch pull {
		case PullDown, PullUp:
			gpioMem[pullReg] |= uint32(pull)
		case PullOff:
			gpioMem[pullReg] &^= 3
		}

		// Wait for value to clock in, this is ugly, sorry :(
		time.Sleep(time.Microsecond)

		gpioMem[pullClkReg] = 1 << shift

		// Wait for value to clock in
		time.Sleep(time.Microsecond)

		gpioMem[pullReg] &^= 3
		gpioMem[pullClkReg] = 0
	}
}

// SetFreq: Set clock speed for given pin in Clock or Pwm mode
//
// Param freq should be in range 4688Hz - 19.2MHz to prevent unexpected behavior,
// however output frequency of Pwm pins can be further adjusted with SetDutyCycle.
// So for smaller frequencies use Pwm pin with large cycle range. (Or implement custom software clock using output pin and sleep.)
//
// Note that some pins share the same clock source, it means that
// changing frequency for one pin will change it also for all pins within a group.
// The groups are:
//   gp_clk0: pins 4, 20, 32, 34
//   gp_clk1: pins 5, 21, 42, 44
//   gp_clk2: pins 6 and 43
//   pwm_clk: pins 12, 13, 18, 19, 40, 41, 45
func SetFreq(pin Pin, freq int) {
	// TODO: would be nice to choose best clock source depending on target frequency, oscilator is used for now
	const sourceFreq = 19200000 // oscilator frequency
	const divMask = 4095        // divi and divf have 12 bits each

	divi := uint32(sourceFreq / freq)
	divf := uint32(((sourceFreq % freq) << 12) / freq)

	divi &= divMask
	divf &= divMask

	clkCtlReg := 28
	clkDivReg := 28
	switch pin {
	case 4, 20, 32, 34: // clk0
		clkCtlReg += 0
		clkDivReg += 1
	case 5, 21, 42, 44: // clk1
		clkCtlReg += 2
		clkDivReg += 3
	case 6, 43: // clk2
		clkCtlReg += 4
		clkDivReg += 5
	case 12, 13, 40, 41, 45, 18, 19: // pwm_clk - shared clk for both pwm channels
		clkCtlReg += 12
		clkDivReg += 13
		StopPwm() // pwm clk busy wont go down without stopping pwm first
		defer StartPwm()
	default:
		return
	}

	mash := uint32(1 << 9) // 1-stage MASH
	if divi < 2 || divf == 0 {
		mash = 0
	}

	memlock.Lock()
	defer memlock.Unlock()

	const PASSWORD = 0x5A000000
	const busy = 1 << 7
	const enab = 1 << 4
	const src = 1 << 0 // oscilator

	clkMem[clkCtlReg] = PASSWORD | (clkMem[clkCtlReg] &^ enab) // stop gpio clock (without changing src or mash)
	for clkMem[clkCtlReg]&busy != 0 {
		time.Sleep(time.Microsecond * 10)
	} // ... and wait for not busy

	clkMem[clkCtlReg] = PASSWORD | mash | src          // set mash and source (without enabling clock)
	clkMem[clkDivReg] = PASSWORD | (divi << 12) | divf // set dividers

	// mash and src can not be changed in same step as enab, to prevent lock-up and glitches
	time.Sleep(time.Microsecond * 10) // ... so wait for them to take effect

	clkMem[clkCtlReg] = PASSWORD | mash | src | enab // finally start clock

	// NOTE without root permission this changes will simply do nothing successfully
}

// SetDutyCycle: Set cycle length (range) and duty length (data) for Pwm pin in M/S mode
//
//   |<- duty ->|
//    __________
//  _/          \_____________/
//   |<------- cycle -------->|
//
// Output frequency is computed as pwm clock frequency divided by cycle length.
// So, to set Pwm pin to freqency 38kHz with duty cycle 1/4, use this combination:
//
//  pin.Pwm()
//  pin.DutyCycle(1, 4)
//  pin.Freq(38000*4)
//
// Note that some pins share common pwm channel,
// so calling this function will set same duty cycle for all pins belonging to channel.
// The channels are:
//   channel 1 (pwm0) for pins 12, 18, 40
//   channel 2 (pwm1) for pins 13, 19, 41, 45.
func SetDutyCycle(pin Pin, dutyLen, cycleLen uint32) {
	const pwmCtlReg = 0
	var (
		pwmDatReg uint
		pwmRngReg uint
		shift     uint // offset inside ctlReg
	)

	switch pin {
	case 12, 18, 40: // channel pwm0
		pwmRngReg = 4
		pwmDatReg = 5
		shift = 0
	case 13, 19, 41, 45: // channel pwm1
		pwmRngReg = 8
		pwmDatReg = 9
		shift = 8
	default:
		return
	}

	const ctlMask = 255 // ctl setting has 8 bits for each channel
	const pwen = 1 << 0 // enable pwm
	const msen = 1 << 7 // use M/S transition instead of pwm algorithm

	// reset settings
	pwmMem[pwmCtlReg] = pwmMem[pwmCtlReg]&^(ctlMask<<shift) | msen<<shift | pwen<<shift
	// set duty cycle
	pwmMem[pwmDatReg] = dutyLen
	pwmMem[pwmRngReg] = cycleLen
	time.Sleep(time.Microsecond * 10)

	// NOTE without root permission this changes will simply do nothing successfully
}

// StopPwm: Stop pwm for both channels
func StopPwm() {
	const pwmCtlReg = 0
	const pwen = 1
	pwmMem[pwmCtlReg] &^= pwen<<8 | pwen
}

// StartPwm starts pwm for both channels
func StartPwm() {
	const pwmCtlReg = 0
	const pwen = 1
	pwmMem[pwmCtlReg] |= pwen<<8 | pwen
}

// EnableIRQs: Enables given IRQs (by setting bit to 1 at intended position).
// See 'ARM peripherals interrupts table' in pheripherals datasheet.
// WARNING: you can corrupt your system, only use this if you know what you are doing.
func EnableIRQs(irqs uint64) {
	const irqEnable1 = 0x210 / 4
	const irqEnable2 = 0x214 / 4
	intrMem[irqEnable1] = uint32(irqs)       // IRQ 0..31
	intrMem[irqEnable2] = uint32(irqs >> 32) // IRQ 32..63
}

// DisableIRQs: Disables given IRQs (by setting bit to 1 at intended position).
// See 'ARM peripherals interrupts table' in pheripherals datasheet.
// WARNING: you can corrupt your system, only use this if you know what you are doing.
func DisableIRQs(irqs uint64) {
	const irqDisable1 = 0x21C / 4
	const irqDisable2 = 0x220 / 4
	intrMem[irqDisable1] = uint32(irqs)       // IRQ 0..31
	intrMem[irqDisable2] = uint32(irqs >> 32) // IRQ 32..63
}

func backupIRQs() {
	const irqEnable1 = 0x210 / 4
	const irqEnable2 = 0x214 / 4
	irqsBackup = uint64(intrMem[irqEnable2])<<32 | uint64(intrMem[irqEnable1])
}

// Open and memory map GPIO memory range from /dev/mem .
// Some reflection magic is used to convert it to a unsafe []uint32 pointer
func Open() (err error) {
	var file *os.File

	// Open fd for rw mem access; try dev/mem first (need root)
	file, err = os.OpenFile("/dev/mem", os.O_RDWR|os.O_SYNC, 0)
	if os.IsPermission(err) { // try gpiomem otherwise (some extra functions like clock and pwm setting wont work)
		file, err = os.OpenFile("/dev/gpiomem", os.O_RDWR|os.O_SYNC, 0)
	}
	if err != nil {
		return
	}
	// FD can be closed after memory mapping
	defer file.Close()

	memlock.Lock()
	defer memlock.Unlock()

	// Memory map GPIO registers to slice
	gpioMem, gpioMem8, err = memMap(file.Fd(), gpioBase)
	if err != nil {
		return
	}

	// Memory map clock registers to slice
	clkMem, clkMem8, err = memMap(file.Fd(), clkBase)
	if err != nil {
		return
	}

	// Memory map pwm registers to slice
	pwmMem, pwmMem8, err = memMap(file.Fd(), pwmBase)
	if err != nil {
		return
	}

	// Memory map spi registers to slice
	spiMem, spiMem8, err = memMap(file.Fd(), spiBase)
	if err != nil {
		return
	}

	// Memory map interruption registers to slice
	intrMem, intrMem8, err = memMap(file.Fd(), intrBase)
	if err != nil {
		return
	}

	backupIRQs() // back up enabled IRQs, to restore it later

	return nil
}

func memMap(fd uintptr, base int64) (mem []uint32, mem8 []byte, err error) {
	mem8, err = syscall.Mmap(
		int(fd),
		base,
		memLength,
		syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_SHARED,
	)
	if err != nil {
		return
	}
	// Convert mapped byte memory to unsafe []uint32 pointer, adjust length as needed
	header := *(*reflect.SliceHeader)(unsafe.Pointer(&mem8))
	header.Len /= (32 / 8) // (32 bit = 4 bytes)
	header.Cap /= (32 / 8)
	mem = *(*[]uint32)(unsafe.Pointer(&header))
	return
}

// Close unmaps GPIO memory
func Close() error {
	EnableIRQs(irqsBackup) // Return IRQs to state where it was before - just to be nice

	memlock.Lock()
	defer memlock.Unlock()
	for _, mem8 := range [][]uint8{gpioMem8, clkMem8, pwmMem8, spiMem8, intrMem8} {
		if err := syscall.Munmap(mem8); err != nil {
			return err
		}
	}
	return nil
}

// Read /proc/device-tree/soc/ranges and determine the base address.
// Use the default Raspberry Pi 1 base address if this fails.
func readBase(offset int64) (int64, error) {
	ranges, err := os.Open("/proc/device-tree/soc/ranges")
	defer ranges.Close()
	if err != nil {
		return 0, err
	}
	b := make([]byte, 4)
	n, err := ranges.ReadAt(b, offset)
	if n != 4 || err != nil {
		return 0, err
	}
	buf := bytes.NewReader(b)
	var out uint32
	err = binary.Read(buf, binary.BigEndian, &out)
	if err != nil {
		return 0, err
	}

	if out == 0 {
		return 0, errors.New("rpio: GPIO base address not found")
	}
	return int64(out), nil
}

func getBase() int64 {
	// Pi 2 & 3 GPIO base address is at offset 4
	b, err := readBase(4)
	if err == nil {
		return b
	}

	// Pi 4 GPIO base address is as offset 8
	b, err = readBase(8)
	if err == nil {
		return b
	}

	// Default to Pi 1
	return int64(bcm2835Base)
}

// The Pi 4 uses a BCM 2711, which has different register offsets and base addresses than the rest of the Pi family (so far).  This
// helper function checks if we're on a 2711 and hence a Pi 4
func isBCM2711() bool {
	return gpioMem[GPPUPPDN3] != 0x6770696f
}
