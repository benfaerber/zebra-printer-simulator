package internal

import (
	"strings"
	"testing"
)

func TestPrinterState_DefaultHealthy(t *testing.T) {
	s := NewPrinterState()
	resp := s.GenerateHSResponse()

	if !strings.Contains(resp, ",0,0,") {
		t.Error("default state should have no error flags")
	}
}

func TestPrinterState_SetError(t *testing.T) {
	s := NewPrinterState()
	s.SetError("paper_out", true)

	resp := s.GenerateHSResponse()
	// Line 1: \x02030,1, ... (paper out = 1)
	if !strings.Contains(resp, "030,1,") {
		t.Errorf("expected paper_out flag in response, got: %q", resp)
	}
}

func TestPrinterState_LabelCount(t *testing.T) {
	s := NewPrinterState()
	if s.LabelCount() != 0 {
		t.Errorf("expected initial label count 0, got %d", s.LabelCount())
	}

	s.IncrementLabelCount()
	s.IncrementLabelCount()

	if s.LabelCount() != 2 {
		t.Errorf("expected label count 2, got %d", s.LabelCount())
	}
}

func TestPrinterState_Snapshot(t *testing.T) {
	s := NewPrinterState()
	s.SetError("head_up", true)

	snap := s.Snapshot()
	if snap["head_up"] != true {
		t.Error("expected head_up=true in snapshot")
	}
	if snap["paper_out"] != false {
		t.Error("expected paper_out=false in snapshot")
	}
}

func TestPrinterState_CanPrint(t *testing.T) {
	s := NewPrinterState()
	if !s.CanPrint() {
		t.Error("fresh printer should be able to print")
	}

	blocking := []string{"paper_out", "head_up", "ribbon_out", "over_temp", "paused"}
	for _, flag := range blocking {
		s := NewPrinterState()
		s.SetError(flag, true)
		if s.CanPrint() {
			t.Errorf("%s should block printing", flag)
		}
		if s.BlockingFault() != flag {
			t.Errorf("expected blocking fault %q, got %q", flag, s.BlockingFault())
		}
	}
}

func TestPrinterState_UnderTempDoesNotBlock(t *testing.T) {
	s := NewPrinterState()
	s.SetError("under_temp", true)
	if !s.CanPrint() {
		t.Error("under_temp is a warning and should not block printing")
	}
}

func TestPrinterState_SGDToggle(t *testing.T) {
	s := NewPrinterState()
	if !s.SGDEnabled() {
		t.Error("SGD should be enabled by default")
	}

	s.SetSGDEnabled(false)
	if s.SGDEnabled() {
		t.Error("SGD should be disabled after toggle")
	}
}
