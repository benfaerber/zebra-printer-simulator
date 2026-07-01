package internal

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/ingridhq/zebrash"
	"github.com/ingridhq/zebrash/drawers"
	"github.com/ingridhq/zebrash/elements"
)

type LabelSize struct {
	WidthMm  float64
	HeightMm float64
}

var (
	LabelSize4x6 = LabelSize{WidthMm: 101.6, HeightMm: 152.4}
	LabelSize6x4 = LabelSize{WidthMm: 152.4, HeightMm: 101.6}
	LabelSize2x4 = LabelSize{WidthMm: 101.6, HeightMm: 50.8}
)

type Renderer struct {
	outputDir string
	labelSize LabelSize
	dpmm      int
	state     *PrinterState
	retention *OutputRetention
	parser    *zebrash.Parser
	drawer    *zebrash.Drawer
	seq       atomic.Uint64
}

type RendererOptions struct {
	OutputDir string
	LabelSize LabelSize
	Dpmm      int
	State     *PrinterState
	Retention *OutputRetention
}

func NewRenderer(opts RendererOptions) *Renderer {
	return &Renderer{
		outputDir: opts.OutputDir,
		labelSize: opts.LabelSize,
		dpmm:      opts.Dpmm,
		state:     opts.State,
		retention: opts.Retention,
		parser:    zebrash.NewParser(),
		drawer:    zebrash.NewDrawer(),
	}
}

// RenderZPL parses a ZPL job and writes a PNG for every label it contains,
// repeated per the job's ^PQ copy count. It returns the paths of all files
// written, in print order.
func (r *Renderer) RenderZPL(data []byte) ([]string, error) {
	labels, err := r.parser.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("parse ZPL: %w", err)
	}

	if len(labels) == 0 {
		return nil, fmt.Errorf("no labels found in ZPL data")
	}

	if err := os.MkdirAll(r.outputDir, 0755); err != nil {
		return nil, fmt.Errorf("create output dir: %w", err)
	}

	size := detectLabelSize(data, r.labelSize, r.dpmm)
	opts := drawers.DrawerOptions{
		LabelWidthMm:  size.WidthMm,
		LabelHeightMm: size.HeightMm,
		Dpmm:          r.dpmm,
	}

	quantity := detectPrintQuantity(data)

	var paths []string
	for _, label := range labels {
		for range quantity {
			if delay := r.state.PrintDelay(); delay > 0 {
				time.Sleep(delay)
			}
			path, err := r.drawLabel(label, opts)
			if err != nil {
				return paths, err
			}
			paths = append(paths, path)
		}
	}

	if r.retention != nil {
		r.retention.Enforce()
	}

	return paths, nil
}

func (r *Renderer) drawLabel(label elements.LabelInfo, opts drawers.DrawerOptions) (string, error) {
	outputPath := filepath.Join(r.outputDir, r.nextFilename())

	f, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	if err := r.drawer.DrawLabelAsPng(label, f, opts); err != nil {
		return "", fmt.Errorf("render PNG: %w", err)
	}
	return outputPath, nil
}

// nextFilename builds a unique, time-ordered PNG name. The monotonic sequence
// suffix keeps names distinct even when many copies render within one
// millisecond.
func (r *Renderer) nextFilename() string {
	n := r.seq.Add(1)
	return fmt.Sprintf("label_%s_%04d.png", time.Now().Format("20060102_150405.000"), n%10000)
}

func (r *Renderer) RenderPreview(data []byte, w io.Writer) error {
	labels, err := r.parser.Parse(data)
	if err != nil {
		return fmt.Errorf("parse ZPL: %w", err)
	}
	if len(labels) == 0 {
		return fmt.Errorf("no labels found in ZPL data")
	}

	size := detectLabelSize(data, r.labelSize, r.dpmm)
	opts := drawers.DrawerOptions{
		LabelWidthMm:  size.WidthMm,
		LabelHeightMm: size.HeightMm,
		Dpmm:          r.dpmm,
	}

	if err := r.drawer.DrawLabelAsPng(labels[0], w, opts); err != nil {
		return fmt.Errorf("render PNG: %w", err)
	}
	return nil
}
