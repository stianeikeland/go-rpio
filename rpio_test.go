package rpio

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	println("Note: bcm pins 2 and 3 has to be directly connected")
	if err := Open(); err != nil {
		panic(err)
	}
	defer Close()
	os.Exit(m.Run())
}

func TestInterrupt(t *testing.T) {
	logIrqRegs(t)
	EnableIRQs(1 << 49)
	EnableIRQs(1 << 50)
	EnableIRQs(1 << 51)
	EnableIRQs(1 << 52)
	logIrqRegs(t)
	DisableIRQs(1 << 49)
	DisableIRQs(1 << 50)
	DisableIRQs(1 << 51)
	DisableIRQs(1 << 52)
	logIrqRegs(t)
	EnableIRQs(irqsBackup)
}

func TestEvent(t *testing.T) {
	src := Pin(3)
	src.Mode(Output)

	pin := Pin(2)
	pin.Mode(Input)
	pin.PullDown()

	t.Run("rising edge", func(t *testing.T) {
		pin.Detect(RiseEdge)
		src.Low()

		for i := 0; ; i++ {
			src.High()

			time.Sleep(time.Second / 10)
			if pin.EdgeDetected() {
				t.Log("edge rised")
			} else {
				t.Errorf("Rise event should be detected")
			}
			if i == 5 {
				break
			}

			src.Low()
		}

		time.Sleep(time.Second / 10)
		if pin.EdgeDetected() {
			t.Error("Rise should not be detected, no change since last call")
		}
		pin.Detect(NoEdge)
		src.High()
		if pin.EdgeDetected() {
			t.Error("Rise should not be detected, events disabled")
		}

	})

	t.Run("falling edge", func(t *testing.T) {
		pin.Detect(FallEdge)
		src.High()

		for i := 0; ; i++ {
			src.Low()

			time.Sleep(time.Second / 10)
			if pin.EdgeDetected() {
				t.Log("edge fallen")
			} else {
				t.Errorf("Fall event should be detected")
			}

			if i == 5 {
				break
			}

			src.High()
		}
		time.Sleep(time.Second / 10)
		if pin.EdgeDetected() {
			t.Error("Fall should not be detected, no change since last call")
		}
		pin.Detect(NoEdge)
		src.Low()
		if pin.EdgeDetected() {
			t.Error("Fall should not be detected, events disabled")
		}
	})

	t.Run("both edges", func(t *testing.T) {
		pin.Detect(AnyEdge)
		src.Low()

		for i := 0; i < 5; i++ {
			src.High()

			if pin.EdgeDetected() {
				t.Log("edge detected")
			} else {
				t.Errorf("Rise event shoud be detected")
			}

			src.Low()

			if pin.EdgeDetected() {
				t.Log("edge detected")
			} else {
				t.Errorf("Fall edge should be detected")
			}
		}

		pin.Detect(NoEdge)
		src.High()
		src.Low()

		if pin.EdgeDetected() {
			t.Errorf("No edge should be detected, events disabled")
		}

	})

	// If this fr⁽eezes your pi,
	// add `dtoverlay=gpio-no-irq` to `/boot/config.txt` and restart your pi,
	// or run as root.
	t.Run("multiple edges", func(t *testing.T) {
		EnableIRQs(15 << 49) // all 4 gpio_int[]
		logIrqRegs(t)
		src.High()
		pin.Detect(FallEdge)

		logIrqRegs(t)

		for i := 0; i < 10000; i++ {
			time.Sleep(time.Millisecond)
			src.High()
			time.Sleep(time.Millisecond)
			src.Low()
		}
		logIrqRegs(t)
		if !pin.EdgeDetected() {
			t.Errorf("Edge not detected")
		}
		logIrqRegs(t)
		pin.Detect(NoEdge)
		logIrqRegs(t)
		EnableIRQs(irqsBackup)
	})

}

func BenchmarkGpio(b *testing.B) {
	src := Pin(3)
	src.Mode(Output)
	src.Low()

	pin := Pin(2)
	pin.Mode(Input)
	pin.PullDown()

	oldWrite := func(pin Pin, state State) {
		p := uint8(pin)

		setReg := p/32 + 7
		clearReg := p/32 + 10

		memlock.Lock()
		defer memlock.Unlock()

		if state == Low {
			gpioMem[clearReg] = 1 << (p & 31)
		} else {
			gpioMem[setReg] = 1 << (p & 31)
		}
	}

	oldToggle := func(pin Pin) {
		switch ReadPin(pin) {
		case Low:
			oldWrite(pin, High)
		case High:
			oldWrite(pin, Low)
		}
	}

	b.Run("write", func(b *testing.B) {
		b.Run("old", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if i%2 == 0 {
					oldWrite(src, High)
				} else {
					oldWrite(src, Low)
				}
			}
		})

		b.Run("new", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if i%2 == 0 {
					WritePin(src, High)
				} else {
					WritePin(src, Low)
				}
			}
		})
	})

	b.Run("toggle", func(b *testing.B) {
		b.Run("old", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				oldToggle(src)
			}
		})

		b.Run("new", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				TogglePin(src)
			}
		})
	})

}

func logIrqRegs(t *testing.T) {
	fmt.Printf("PENDING(% X) FIQ(% X) ENAB(% X) DISAB(% X)\n",
		intrMem8[0x200:0x20C],
		intrMem8[0x20C:0x210],
		intrMem8[0x210:0x21C],
		intrMem8[0x21C:0x228],
	)
}
