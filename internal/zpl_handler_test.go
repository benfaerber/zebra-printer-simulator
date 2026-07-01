package internal

import (
	"testing"
)

func TestClassifyInput_HS(t *testing.T) {
	if ClassifyInput("~HS") != CommandHS {
		t.Error("expected CommandHS")
	}
	if ClassifyInput("  ~HS  ") != CommandHS {
		t.Error("expected CommandHS with whitespace")
	}
}

func TestClassifyInput_SGD(t *testing.T) {
	cmd := `! U1 getvar "odometer.total_label_count"`
	if ClassifyInput(cmd) != CommandSGD {
		t.Error("expected CommandSGD")
	}
}

func TestClassifyInput_ZPL(t *testing.T) {
	if ClassifyInput("^XA^FO50,50^ADN,36,20^FDTest^FS^XZ") != CommandZPL {
		t.Error("expected CommandZPL")
	}
}

func TestClassifyInput_Unknown(t *testing.T) {
	if ClassifyInput("hello") != CommandUnknown {
		t.Error("expected CommandUnknown")
	}
}

func newTestSGD(state *PrinterState) *SGDResponder {
	return NewSGDResponder(state, 203, 812, "Test Printer")
}

func TestSGD_LabelCount(t *testing.T) {
	state := NewPrinterState()
	state.IncrementLabelCount()
	state.IncrementLabelCount()

	resp := newTestSGD(state).Handle(`! U1 getvar "odometer.total_label_count"`)
	if resp != "\"2\"\r\n" {
		t.Errorf("expected label count 2, got %q", resp)
	}
}

func TestSGD_Disabled(t *testing.T) {
	state := NewPrinterState()
	state.SetSGDEnabled(false)

	resp := newTestSGD(state).Handle(`! U1 getvar "odometer.total_label_count"`)
	if resp != "?\r\n" {
		t.Errorf("expected ? when SGD disabled, got %q", resp)
	}
}

func TestSGD_UnknownVar(t *testing.T) {
	resp := newTestSGD(NewPrinterState()).Handle(`! U1 getvar "some.unknown.var"`)
	if resp != "?\r\n" {
		t.Errorf("expected ? for unknown var, got %q", resp)
	}
}

func TestSGD_ResolutionAndWidthFromConfig(t *testing.T) {
	sgd := newTestSGD(NewPrinterState())
	if got := sgd.Handle(`! U1 getvar "head.resolution.in_dpi"`); got != "\"203\"\r\n" {
		t.Errorf("expected dpi 203, got %q", got)
	}
	if got := sgd.Handle(`! U1 getvar "ezpl.print_width"`); got != "\"812\"\r\n" {
		t.Errorf("expected print width 812, got %q", got)
	}
}

func TestSGD_SetvarAcceptedSilently(t *testing.T) {
	resp := newTestSGD(NewPrinterState()).Handle(`! U1 setvar "media.darkness" "15"`)
	if resp != "" {
		t.Errorf("expected empty response for setvar, got %q", resp)
	}
}
