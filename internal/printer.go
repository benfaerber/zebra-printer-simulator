package internal

import (
	"log/slog"
	"sync"
)

// Printer turns submitted ZPL jobs into rendered labels. It owns the printer's
// serial behavior: only one job renders at a time, and jobs that arrive while a
// blocking fault is active are held until the fault clears.
type Printer struct {
	state    *PrinterState
	renderer *Renderer
	webhook  Webhook
	events   *EventHub

	mu   sync.Mutex
	held [][]byte
}

type PrinterOptions struct {
	State    *PrinterState
	Renderer *Renderer
	Webhook  Webhook
	Events   *EventHub
}

func NewPrinter(opts PrinterOptions) *Printer {
	if opts.Webhook == nil {
		opts.Webhook = NoopWebhook{}
	}
	return &Printer{
		state:    opts.State,
		renderer: opts.Renderer,
		webhook:  opts.Webhook,
		events:   opts.Events,
	}
}

// Submit renders a ZPL job now, or holds it if the printer cannot currently
// print. Held jobs render later via Flush.
func (p *Printer) Submit(data []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.state.CanPrint() {
		p.hold(data)
		return
	}
	p.render(data)
}

// Flush renders held jobs in arrival order for as long as the printer can
// print. It is called when a fault clears so queued work resumes.
func (p *Printer) Flush() {
	p.mu.Lock()
	defer p.mu.Unlock()

	rendered := false
	for len(p.held) > 0 && p.state.CanPrint() {
		next := p.held[0]
		p.held = p.held[1:]
		p.render(next)
		rendered = true
	}
	if rendered {
		p.state.SetFormatsInBuffer(len(p.held))
	}
}

// DiscardHeld drops any queued jobs without printing them, used on reset.
func (p *Printer) DiscardHeld() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.held = nil
	p.state.SetFormatsInBuffer(0)
}

// HeldCount reports how many jobs are waiting for a fault to clear.
func (p *Printer) HeldCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.held)
}

func (p *Printer) hold(data []byte) {
	p.held = append(p.held, data)
	p.state.SetFormatsInBuffer(len(p.held))
	slog.Info("job held", "reason", p.state.BlockingFault(), "queued", len(p.held))
	p.notify()
}

func (p *Printer) render(data []byte) {
	paths, err := p.renderer.RenderZPL(data)
	if err != nil {
		slog.Warn("render failed", "err", err)
		p.state.IncrementRenderFailures()
		p.notify()
		return
	}

	for _, path := range paths {
		p.state.IncrementLabelCount()
		slog.Info("rendered label", "path", path, "label_count", p.state.LabelCount())
		p.webhook.Notify(eventFromPath(path, p.state.LabelCount()))
	}
	p.notify()
}

func (p *Printer) notify() {
	if p.events != nil {
		p.events.Publish()
	}
}
