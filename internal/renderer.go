package internal

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/ingridhq/zebrash"
	"github.com/ingridhq/zebrash/drawers"
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
	outputDir  string
	labelSize  LabelSize
	dpmm       int
	printDelay time.Duration
	retention  *OutputRetention
	parser     *zebrash.Parser
	drawer     *zebrash.Drawer
}

type RendererOptions struct {
	OutputDir  string
	LabelSize  LabelSize
	Dpmm       int
	PrintDelay time.Duration
	Retention  *OutputRetention
}

func NewRenderer(opts RendererOptions) *Renderer {
	return &Renderer{
		outputDir:  opts.OutputDir,
		labelSize:  opts.LabelSize,
		dpmm:       opts.Dpmm,
		printDelay: opts.PrintDelay,
		retention:  opts.Retention,
		parser:     zebrash.NewParser(),
		drawer:     zebrash.NewDrawer(),
	}
}

func (r *Renderer) RenderZPL(data []byte) (string, error) {
	if r.printDelay > 0 {
		time.Sleep(r.printDelay)
	}

	labels, err := r.parser.Parse(data)
	if err != nil {
		return "", fmt.Errorf("parse ZPL: %w", err)
	}

	if len(labels) == 0 {
		return "", fmt.Errorf("no labels found in ZPL data")
	}

	if err := os.MkdirAll(r.outputDir, 0755); err != nil {
		return "", fmt.Errorf("create output dir: %w", err)
	}

	filename := fmt.Sprintf("label_%s.png", time.Now().Format("20060102_150405.000"))
	outputPath := filepath.Join(r.outputDir, filename)

	f, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	size := detectLabelSize(data, r.labelSize, r.dpmm)
	opts := drawers.DrawerOptions{
		LabelWidthMm:  size.WidthMm,
		LabelHeightMm: size.HeightMm,
		Dpmm:          r.dpmm,
	}

	if err := r.drawer.DrawLabelAsPng(labels[0], f, opts); err != nil {
		return "", fmt.Errorf("render PNG: %w", err)
	}

	if r.retention != nil {
		r.retention.Enforce()
	}

	return outputPath, nil
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
