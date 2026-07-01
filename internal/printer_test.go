package internal

import (
	"testing"
)

const sampleZPL = "^XA^FO50,50^A0N,30,30^FDhello^FS^XZ"

func newTestPrinter(t *testing.T) (*Printer, *PrinterState, string) {
	t.Helper()
	dir := t.TempDir()
	state := NewPrinterState()
	renderer := NewRenderer(RendererOptions{
		OutputDir: dir,
		LabelSize: LabelSize4x6,
		Dpmm:      8,
		State:     state,
	})
	printer := NewPrinter(PrinterOptions{State: state, Renderer: renderer})
	return printer, state, dir
}

func TestPrinter_RendersWhenReady(t *testing.T) {
	printer, state, dir := newTestPrinter(t)

	printer.Submit([]byte(sampleZPL))

	if state.LabelCount() != 1 {
		t.Errorf("expected 1 label, got %d", state.LabelCount())
	}
	if countPNGs(t, dir) != 1 {
		t.Errorf("expected 1 PNG on disk, got %d", countPNGs(t, dir))
	}
}

func TestPrinter_HoldsJobWhilePaperOut(t *testing.T) {
	printer, state, dir := newTestPrinter(t)
	state.SetError("paper_out", true)

	printer.Submit([]byte(sampleZPL))

	if state.LabelCount() != 0 {
		t.Errorf("expected no labels while paper out, got %d", state.LabelCount())
	}
	if printer.HeldCount() != 1 {
		t.Errorf("expected 1 held job, got %d", printer.HeldCount())
	}
	if countPNGs(t, dir) != 0 {
		t.Errorf("expected no PNGs while held, got %d", countPNGs(t, dir))
	}
}

func TestPrinter_FlushesHeldJobsWhenFaultClears(t *testing.T) {
	printer, state, dir := newTestPrinter(t)
	state.SetError("paper_out", true)
	printer.Submit([]byte(sampleZPL))
	printer.Submit([]byte(sampleZPL))

	state.SetError("paper_out", false)
	printer.Flush()

	if printer.HeldCount() != 0 {
		t.Errorf("expected held queue drained, got %d", printer.HeldCount())
	}
	if state.LabelCount() != 2 {
		t.Errorf("expected 2 labels after flush, got %d", state.LabelCount())
	}
	if countPNGs(t, dir) != 2 {
		t.Errorf("expected 2 PNGs after flush, got %d", countPNGs(t, dir))
	}
}

func TestPrinter_DiscardHeld(t *testing.T) {
	printer, state, _ := newTestPrinter(t)
	state.SetError("paused", true)
	printer.Submit([]byte(sampleZPL))

	printer.DiscardHeld()

	if printer.HeldCount() != 0 {
		t.Errorf("expected held queue cleared, got %d", printer.HeldCount())
	}
	if state.FormatsInBuffer() != 0 {
		t.Errorf("expected formats_in_buffer reset, got %d", state.FormatsInBuffer())
	}
}
