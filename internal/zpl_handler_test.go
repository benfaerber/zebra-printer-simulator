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

func TestHandleSGDCommand_LabelCount(t *testing.T) {
	state := NewPrinterState()
	state.IncrementLabelCount()
	state.IncrementLabelCount()

	resp := HandleSGDCommand(`! U1 getvar "odometer.total_label_count"`, state)
	if resp != "\"2\"\r\n" {
		t.Errorf("expected label count 2, got %q", resp)
	}
}

func TestHandleSGDCommand_Disabled(t *testing.T) {
	state := NewPrinterState()
	state.SetSGDEnabled(false)

	resp := HandleSGDCommand(`! U1 getvar "odometer.total_label_count"`, state)
	if resp != "?\r\n" {
		t.Errorf("expected ? when SGD disabled, got %q", resp)
	}
}

func TestHandleSGDCommand_UnknownVar(t *testing.T) {
	state := NewPrinterState()
	resp := HandleSGDCommand(`! U1 getvar "some.unknown.var"`, state)
	if resp != "?\r\n" {
		t.Errorf("expected ? for unknown var, got %q", resp)
	}
}
