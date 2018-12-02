go-rpio
=======

Native GPIO-Gophers for your Pi!

**Documentation:** [![GoDoc](https://godoc.org/github.com/stianeikeland/go-rpio?status.svg)](https://godoc.org/github.com/stianeikeland/go-rpio)

go-rpio is a Go library for accessing [GPIO](http://elinux.org/Rpi_Low-level_peripherals)-pins
on the [Raspberry Pi](https://en.wikipedia.org/wiki/Raspberry_Pi).

It requires no external c libraries such as
[WiringPI](https://projects.drogon.net/raspberry-pi/wiringpi/) or [bcm2835](http://www.open.com.au/mikem/bcm2835).

There's a tiny bit of additional information over at my [blog](https://blog.eikeland.se/2013/07/30/go-gpio-library-for-raspberry-pi/).

![raspberrypi-blink](http://stianeikeland.files.wordpress.com/2013/07/animated.gif)

## Releases ##
- 1.0.0 - Supports original rpi A/B/B+
- 2.0.0 - Adds support for rpi 2, by @akramer
- 3.0.0 - Adds support for /dev/gpiomem, by @dotdoom
- 4.0.0 - Adds support for PWM and Clock modes, by @Drahoslav7
- 4.1.0 - Adds support for edge detection, by @Drahoslav7
- 4.2.0 - Faster write and toggle of output pins, by @Drahoslav7
- 4.3.0 - Adds support for SPI, by @Drahoslav7
- 4.4.0 - Support for disabling interrupts (workaround for #35), by @Drahoslav7

## Usage ##

```go
import "github.com/stianeikeland/go-rpio/v4"
```

If you're using an older go.mod incompatible you should instead use:

```go
import "github.com/stianeikeland/go-rpio"
```

Open memory range for GPIO access in /dev/mem

```go
err := rpio.Open()
```

Initialize a pin, run basic operations.
Pin refers to the bcm2835 pin, not the physical pin on the raspberry pi header. Pin 10 here is exposed on the pin header as physical pin 19.

```go
pin := rpio.Pin(10)

pin.Output()       // Output mode
pin.High()         // Set pin High
pin.Low()          // Set pin Low
pin.Toggle()       // Toggle pin (Low -> High -> Low)

pin.Input()        // Input mode
res := pin.Read()  // Read state from pin (High / Low)

pin.Mode(rpio.Output)   // Alternative syntax
pin.Write(rpio.High)    // Alternative syntax
```

Pull up/down/off can be set using:

```go
pin.PullUp()
pin.PullDown()
pin.PullOff()

pin.Pull(rpio.PullUp)
```

Unmap memory when done

```go
rpio.Close()
```

Also see example [examples/blinker/blinker.go](examples/blinker/blinker.go)

### SPI

#### setup/teardown
  - `rpio.SpiBegin(rpio.Spi0)` has to be called first before using any Spi func. It will change pin modes to `Spi` and initialize default setting.
  - `rpio.SpiEnd(rpio.Spi0)` should be called at the end, it will switch pin modes to `Input`.

#### transferring data
  - `rpio.SpiTransmit(byte)` or `rpio.SpiTransmit(bytes...)` will transmit byte or bytes to slave.
  - `rpio.SpiReceive(n)` will return n bytes received from slave.
  - `rpio.SpiExchange(buffer)` will simultaneously transmit data from the buffer to slave and data from slave to the same buffer in full duplex way.

#### settings
  - `rpio.SpiSpeed(hz)` will set transmit speed of SPI
  - `rpio.SpiChipSelect(n)` will select chip/slave (ce0, ce1, or ce2) to which transferring will be done
  - `rpio.SpiChipSelectPolarity(n, pol)` set chip select polarity (low enabled is used by default which usually works most of the time)
  - `rpio.SpiMode(cpol, cpha)` set clock/communication mode (=combination of clock polarity and clock phase; cpol=0, cpha=0 is used by default which usually works most of the time)

## Other ##

Currently, it supports basic functionality such as:
- Pin Direction (Input / Output)
- Write (High / Low)
- Read (High / Low)
- Pull (Up / Down / Off)
- PWM (hardware, on supported pins)
- Clock
- Edge detection

It works by memory-mapping the bcm2835 gpio range, and therefore require root/administrative-rights to run.

## Using without root ##

This library can utilize the new [/dev/gpiomem](https://github.com/raspberrypi/linux/pull/1112/files) 
memory range if available. 

You will probably need to upgrade to the latest kernel (or wait for the next raspbian release) if you're missing /dev/gpiomem. You will also need to add a `gpio` group, add your user to the group, and then set up udev rules. I would recommend using [create_gpio_user_permissions.py](https://github.com/waveform80/rpi-gpio/blob/master/create_gpio_user_permissions.py) if you're unsure how to do this.

PWM modes will still require root.

