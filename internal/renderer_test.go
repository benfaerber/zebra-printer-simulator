package internal

import (
	"testing"
)

func newTestRenderer(t *testing.T) (*Renderer, *PrinterState) {
	t.Helper()
	state := NewPrinterState()
	renderer := NewRenderer(RendererOptions{
		OutputDir: t.TempDir(),
		LabelSize: LabelSize4x6,
		Dpmm:      8,
		State:     state,
	})
	return renderer, state
}

func TestRenderZPL_SingleLabel(t *testing.T) {
	renderer, _ := newTestRenderer(t)

	paths, err := renderer.RenderZPL([]byte(sampleZPL))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if len(paths) != 1 {
		t.Errorf("expected 1 path, got %d", len(paths))
	}
}

func TestRenderZPL_HonorsPrintQuantity(t *testing.T) {
	renderer, _ := newTestRenderer(t)
	zpl := "^XA^FO50,50^A0N,30,30^FDcopies^FS^PQ3^XZ"

	paths, err := renderer.RenderZPL([]byte(zpl))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if len(paths) != 3 {
		t.Errorf("expected 3 copies from ^PQ3, got %d", len(paths))
	}
	if len(unique(paths)) != 3 {
		t.Errorf("expected 3 distinct filenames, got %d", len(unique(paths)))
	}
}

func TestRenderZPL_QuantityClamped(t *testing.T) {
	renderer, _ := newTestRenderer(t)
	zpl := "^XA^FO50,50^A0N,30,30^FDlots^FS^PQ9999^XZ"

	paths, err := renderer.RenderZPL([]byte(zpl))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if len(paths) != maxPrintQuantity {
		t.Errorf("expected quantity clamped to %d, got %d", maxPrintQuantity, len(paths))
	}
}

func unique(items []string) map[string]struct{} {
	set := make(map[string]struct{}, len(items))
	for _, item := range items {
		set[item] = struct{}{}
	}
	return set
}
