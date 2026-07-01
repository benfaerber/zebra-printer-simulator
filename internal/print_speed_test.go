package internal

import (
	"testing"
)

func TestParsePrintSpeed(t *testing.T) {
	for _, name := range []string{"fast", "normal", "slow"} {
		speed, ok := ParsePrintSpeed(name)
		if !ok {
			t.Errorf("expected %q to parse as a valid speed", name)
		}
		if string(speed) != name {
			t.Errorf("expected speed %q, got %q", name, speed)
		}
	}

	if _, ok := ParsePrintSpeed("turbo"); ok {
		t.Error("expected unknown speed to be rejected")
	}
}

func TestPrintSpeedDelayOrdering(t *testing.T) {
	if PrintSpeedFast.Delay() != 0 {
		t.Errorf("expected fast to have no delay, got %v", PrintSpeedFast.Delay())
	}
	if !(PrintSpeedFast.Delay() < PrintSpeedNormal.Delay() &&
		PrintSpeedNormal.Delay() < PrintSpeedSlow.Delay()) {
		t.Error("expected delays to increase from fast to slow")
	}
}

func TestPrinterStateDefaultSpeed(t *testing.T) {
	s := NewPrinterState()
	if s.PrintSpeed() != DefaultPrintSpeed {
		t.Errorf("expected default speed %q, got %q", DefaultPrintSpeed, s.PrintSpeed())
	}
	if s.PrintDelay() != DefaultPrintSpeed.Delay() {
		t.Errorf("expected default delay %v, got %v", DefaultPrintSpeed.Delay(), s.PrintDelay())
	}
}

func TestPrinterStateResetRestoresSpeed(t *testing.T) {
	s := NewPrinterState()
	s.SetPrintSpeed(PrintSpeedSlow)
	s.Reset()
	if s.PrintSpeed() != DefaultPrintSpeed {
		t.Errorf("expected reset to restore default speed, got %q", s.PrintSpeed())
	}
}
