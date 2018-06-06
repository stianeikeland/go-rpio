package rpio

import (
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
}
