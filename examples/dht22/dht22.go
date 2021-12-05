package main

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/stianeikeland/go-rpio/v4"
)

func main() {
	err := rpio.Open()
	if err != nil {
		log.Panicf("Could not open GPIO: %s", err)
	}
	// change the pin number to the BCM pin where your DHT22 is attached
	log.Println(readDHT22(rpio.Pin(24)))
}

func readDHT22(pin rpio.Pin) (float32, float32, error) {
	// let data line be pulled high by pull-up
	pin.Input()
	time.Sleep(1000 * time.Microsecond)
	// pull data line low for 1100 micros to signal ready to read
	pin.Output()
	pin.Low()
	time.Sleep(1100 * time.Microsecond)
	// leave data line floating again, sensor writes data now
	pin.Input()
	pos := 0
	var now int64
	level := rpio.Low
	lastChange := time.Now().UnixMicro()
	syncCycles, dataCycles := make([]uint16, 50), make([]uint16, 50)
	// wait for incoming pulses from sensor until buffer is full or timeout is reached
	for {
		now = time.Now().UnixMicro()
		if pin.Read() != level {
			if level == rpio.Low {
				level = rpio.High
				// level changed to HIGH, sync cycle
				syncCycles[pos] = uint16(now - lastChange)

			} else {
				level = rpio.Low
				// level changed to LOW, data cycle
				dataCycles[pos] = uint16(now - lastChange)
				// increment position
				pos++
				if pos >= 50 {
					// buffer is full, stop reading
					break
				}
			}
			lastChange = now
		} else if now-lastChange >= 8000 {
			// pin does not change anymore, stop reading
			break
		}
	}
	// we need at least 40 pulses for a valid data packet
	if pos < 40 {
		return 0, 0, fmt.Errorf("timeout: %d packets received", pos)
	}
	// calculate average sync pulse duration
	offset := pos - 40
	var syncAverage float32 = 0
	for i := offset; i < 40; i++ {
		syncAverage += float32(syncCycles[i])
	}
	syncAverage /= 40
	// extract data bits
	data := make([]uint8, 5)
	for i := 0; i < 40; i++ {
		if dataCycles[i+offset] > uint16(syncAverage) {
			data[i/8] |= 1 << (7 - i%8)
		}
	}
	// verify checksum
	if data[4] != ((data[0] + data[1] + data[2] + data[3]) & 0xFF) {
		return 0, 0, errors.New("checksum error")
	}
	// calculate temperature and humidity
	var tmp, hum float32
	tmp = float32(uint16(data[2]&0x7F)<<8|uint16(data[3])) * 0.1
	hum = float32(uint16(data[0])<<8|uint16(data[1])) * 0.1
	if tmp < -40 || tmp > 80 || hum < 0 || hum > 100 {
		return 0, 0, errors.New("out of range")
	}
	return tmp, hum, nil
}
