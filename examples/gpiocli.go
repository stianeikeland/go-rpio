package main

import (
	"flag"
	"fmt"
	rpio "github.com/stianeikeland/go-rpio"
	"os"
	"runtime"
)

func main() {
	var pin = flag.Int("p", 0, " number of pin to use")
	var high = flag.Bool("h", false, " return 0 if value is HIGH, omit to set value")
	var low = flag.Bool("l", false, " return 0 if value is LOW, omit to set value")
	var set = flag.Int("s", 0, " set pin value to high(1) or low(-1)")
	flag.Parse()

	if os.Geteuid() != 0 {
		fmt.Println("This program have to be run as root, or  SUID/GUID set to  0 on execution!")
		os.Exit(1)
	}

	if runtime.GOARCH != "arm" {
		fmt.Println("This program can be unpredictable on other machines rather than Raspberry Pi")
		os.Exit(1)
	}

	if runtime.GOOS != "linux" {
		fmt.Println("This program have to be executed on linux only!")
		os.Exit(1)
	}

	// Open and map memory to access gpio, check for errors
	if err := rpio.Open(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Unmap gpio memory when done
	defer rpio.Close()

	if pin == 0 {
		fmt.Println("Set the pin number to use!")
		os.Exit(1)
	} else {
		if high || low {
			//we get the value for pin
			pin := rpio.Pin(pin)
			pin.Input() // Input mode
			res := pin.Read()

			if high {
				if res == rpio.High {
					os.Exit(0)
				} else {
					os.Exit(1)
				}
			}

			if low {
				if res == rpio.Low {
					os.Exit(0)
				} else {
					os.Exit(1)
				}
			}
			os.Exit(1)
		} else {
			switch set {
			case 1:
				pin := rpio.Pin(pin)
				pin.Output() // Output mode
				pin.High()
				os.Exit(0)
				break
			case -1:
				//we set the value for pin
				pin := rpio.Pin(pin)
				pin.Output() // Output mode
				pin.Low()
				os.Exit(0)
				break
			default:
				flag.PrintDefaults()
				os.Exit(1)
			}
		}
	}
}
