package rpio

import (
	"fmt"
	"os"
	"reflect"
	"sync"
	"syscall"
	"unsafe"
)

type Direction uint8
type Pin uint8
type State uint8

// Memory offsets for gpio, see the spec for more details
// http://www.raspberrypi.org/wp-content/uploads/2012/02/BCM2835-ARM-Peripherals.pdf
const (
	bcm2835Base = 0x20000000
	gpioBase    = bcm2835Base + 0x200000

	pinmask uint32 = 7 // 0b111
)

// Pin modes (input, output)
const (
	INPUT Direction = iota
	OUTPUT
)

// State of pin, high/low
const (
	LOW State = iota
	HIGH
)

var (
	mem     []uint32
	mem8    []uint8
	memlock sync.Mutex
)

func (pin Pin) High() {
	WritePin(uint8(pin), HIGH)
}

func (pin Pin) Low() {
	WritePin(uint8(pin), LOW)
}

func (pin Pin) Mode(dir Direction) {
	PinMode(uint8(pin), dir)
}

func (pin Pin) Write(state State) {
	WritePin(uint8(pin), state)
}

func (pin Pin) Read() State {
	return ReadPin(uint8(pin))
}

func PinMode(pin uint8, direction Direction) {
	fsel := pin / 10
	shift := (pin % 10) * 3

	fmt.Println(pin, fsel, shift)
	fmt.Printf("0b%b  \n", mem[fsel])

	memlock.Lock()

	if direction == INPUT {
		fmt.Printf("0b%b\n\n", mem[fsel]&^(pinmask<<shift))
	} else {
		fmt.Printf("0b%b\n\n", mem[fsel]&^(pinmask<<shift)|(1<<shift))
	}

	memlock.Unlock()
}

func WritePin(pin uint8, state State) {

	clearReg := pin/32 + 10
	setReg := pin/32 + 7

	memlock.Lock()

	if state == LOW {
		fmt.Printf("0b%b  \n", mem[clearReg])
		fmt.Printf("0b%b\n\n", 1<<(pin&31))
	} else {
		fmt.Printf("0b%b  \n", mem[setReg])
		fmt.Printf("0b%b\n\n", 1<<(pin&31))
	}

	memlock.Unlock()
}

func ReadPin(pin uint8) State {
	// input level register offset
	levelReg := pin/32 + 13

	//memlock.Lock()
	fmt.Printf("0b%b  \n", mem[levelReg])
	fmt.Printf("0b%b  \n", mem[levelReg]&(1<<pin))

	if (mem[levelReg] & (1 << pin)) != 0 {
		return HIGH
	}

	return LOW
}

func Open() (err error) {
	var file *os.File

	// Open fd for rw mem access
	file, err = os.OpenFile("/dev/mem", os.O_RDWR|os.O_SYNC, 0)
	if err != nil {
		return
	}

	// Can be closed after memory mapping
	defer file.Close()

	memlock.Lock()

	// Memory map GPIO registers to byte array
	mem8, err = syscall.Mmap(int(file.Fd()), gpioBase, 4*1024, syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return
	}

	// Convert mapped byte memory to unsafe []uint32 pointer, adjust length as needed
	header := *(*reflect.SliceHeader)(unsafe.Pointer(&mem8))
	header.Len /= (32 / 8) // (32 bit = 4 bytes)
	header.Cap /= (32 / 8)

	mem = *(*[]uint32)(unsafe.Pointer(&header))

	memlock.Unlock()

	return nil
}

func Close() {
	// Unmap memory
	memlock.Lock()
	syscall.Munmap(mem8)
	memlock.Unlock()
}
