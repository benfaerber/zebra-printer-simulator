package internal

import (
	"fmt"
	"sync"
)

type PrinterState struct {
	mu              sync.RWMutex
	paperOut        bool
	paused          bool
	headUp          bool
	ribbonOut       bool
	underTemp       bool
	overTemp        bool
	labelCount      int
	formatsInBuffer int
	renderFailures  int
	sgdEnabled      bool
}

func NewPrinterState() *PrinterState {
	return &PrinterState{sgdEnabled: true}
}

func (s *PrinterState) SetError(flag string, enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch flag {
	case "paper_out":
		s.paperOut = enabled
	case "paused":
		s.paused = enabled
	case "head_up":
		s.headUp = enabled
	case "ribbon_out":
		s.ribbonOut = enabled
	case "under_temp":
		s.underTemp = enabled
	case "over_temp":
		s.overTemp = enabled
	}
}

func (s *PrinterState) SetSGDEnabled(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sgdEnabled = enabled
}

func (s *PrinterState) SGDEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sgdEnabled
}

func (s *PrinterState) IncrementLabelCount() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.labelCount++
}

func (s *PrinterState) LabelCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.labelCount
}

func (s *PrinterState) SetFormatsInBuffer(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.formatsInBuffer = n
}

func (s *PrinterState) FormatsInBuffer() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.formatsInBuffer
}

func (s *PrinterState) IncrementRenderFailures() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.renderFailures++
}

func (s *PrinterState) RenderFailures() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.renderFailures
}

func (s *PrinterState) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.paperOut = false
	s.paused = false
	s.headUp = false
	s.ribbonOut = false
	s.underTemp = false
	s.overTemp = false
	s.labelCount = 0
	s.formatsInBuffer = 0
	s.renderFailures = 0
	s.sgdEnabled = true
}

func (s *PrinterState) GenerateHSResponse() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	boolToInt := func(b bool) int {
		if b {
			return 1
		}
		return 0
	}

	line1 := fmt.Sprintf("\x02030,%d,%d,0626,%03d,0,0,0,000,0,%d,%d\x03",
		boolToInt(s.paperOut),
		boolToInt(s.paused),
		s.formatsInBuffer,
		boolToInt(s.underTemp),
		boolToInt(s.overTemp),
	)

	line2 := fmt.Sprintf("\x02001,0,%d,%d,1,2,0416,0,00000000,1,000\x03",
		boolToInt(s.headUp),
		boolToInt(s.ribbonOut),
	)

	line3 := "\x020000,0\x03"

	return line1 + line2 + line3
}

func (s *PrinterState) Snapshot() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return map[string]interface{}{
		"paper_out":         s.paperOut,
		"paused":            s.paused,
		"head_up":           s.headUp,
		"ribbon_out":        s.ribbonOut,
		"under_temp":        s.underTemp,
		"over_temp":         s.overTemp,
		"label_count":       s.labelCount,
		"formats_in_buffer": s.formatsInBuffer,
		"render_failures":   s.renderFailures,
		"sgd_enabled":       s.sgdEnabled,
	}
}

func (s *PrinterState) MetricsSnapshot() MetricsSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return MetricsSnapshot{
		LabelCount:      s.labelCount,
		FormatsInBuffer: s.formatsInBuffer,
		RenderFailures:  s.renderFailures,
		SGDEnabled:      s.sgdEnabled,
		Faults: map[string]bool{
			"paper_out":  s.paperOut,
			"paused":     s.paused,
			"head_up":    s.headUp,
			"ribbon_out": s.ribbonOut,
			"under_temp": s.underTemp,
			"over_temp":  s.overTemp,
		},
	}
}

type MetricsSnapshot struct {
	LabelCount      int
	FormatsInBuffer int
	RenderFailures  int
	SGDEnabled      bool
	Faults          map[string]bool
}
