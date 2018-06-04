package rpio

import (
	"testing"
	"time"
	"os"
)

func TestMain(m *testing.M) {
	if err := Open(); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

func TestRisingEdgeEvent(t *testing.T) {
	// bcm pins 2 and 3 has to be directly connected
	src := Pin(3)
	src.Mode(Output)
	src.Low()

	pin := Pin(2)
	pin.Mode(Input)
	pin.PullDown()
	pin.Detect(RiseEdge)

	timeout := time.After(time.Second)
	loop: for {
		src.High()

		time.Sleep(time.Second/5)
		if pin.EdgeDetected() {
			t.Log("edge rised")
		} else {
			t.Errorf("Raise event should be detected")
		}
		select {
			case <- timeout:
				break loop
			default:
		}

		src.Low()
	}
	if pin.EdgeDetected() {
		t.Error("Rise should not be detected, no change since last call")
	}
	pin.Detect(NoEdge)
	src.High()
	if pin.EdgeDetected() {
		t.Error("Fall should not be detected, events disabled")
	}
}

func TestFallingEdgeEvent(t *testing.T) {
	// bcm pins 2 and 3 has to be directly connected
	src := Pin(3)
	src.Mode(Output)
	src.High()

	pin := Pin(2)
	pin.Mode(Input)
	pin.PullDown()
	pin.Detect(FallEdge)

	timeout := time.After(time.Second)
	loop: for {
			src.Low()

			time.Sleep(time.Second/5)
			if pin.EdgeDetected() {
				t.Log("edge fallen")
			} else {
				t.Errorf("Fall event should be detected")
			}

			select {
				case <- timeout:
					break loop
				default:
			}

			src.High()
	}
	if pin.EdgeDetected() {
		t.Error("Fall should not be detected, no change since last call")
	}
	pin.Detect(NoEdge)
	src.Low()
	if pin.EdgeDetected() {
		t.Error("Fall should not be detected, events disabled")
	}
}