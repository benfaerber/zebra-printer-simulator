package internal

import (
	"fmt"
	"strings"
)

// firmwareVersion is the value reported for appl.name, mimicking a Zebra
// firmware string closely enough for clients that sniff it.
const firmwareVersion = "V1.0.0-SIM"

// SGDResponder answers Set-Get-Do queries (`! U1 getvar ...`). It reports live
// state from the printer plus static device facts supplied at construction, so
// values such as resolution and print width match the simulator's configured
// DPI and label size instead of being hardcoded.
type SGDResponder struct {
	state        *PrinterState
	dpi          int
	printWidth   int
	friendlyName string
}

func NewSGDResponder(state *PrinterState, dpi, printWidth int, friendlyName string) *SGDResponder {
	return &SGDResponder{
		state:        state,
		dpi:          dpi,
		printWidth:   printWidth,
		friendlyName: friendlyName,
	}
}

// Handle returns the wire response for an SGD command. Unknown vars yield the
// Zebra "?" reply; setvar commands are accepted silently, as real printers do.
func (r *SGDResponder) Handle(data string) string {
	if !r.state.SGDEnabled() {
		return sgdReply("?")
	}

	if strings.Contains(data, "setvar") {
		return ""
	}

	switch {
	case strings.Contains(data, `"odometer.total_label_count"`):
		return sgdReply(fmt.Sprintf("%d", r.state.LabelCount()))
	case strings.Contains(data, `"head.resolution.in_dpi"`):
		return sgdReply(fmt.Sprintf("%d", r.dpi))
	case strings.Contains(data, `"ezpl.print_width"`):
		return sgdReply(fmt.Sprintf("%d", r.printWidth))
	case strings.Contains(data, `"device.friendly_name"`):
		return sgdReply(r.friendlyName)
	case strings.Contains(data, `"appl.name"`):
		return sgdReply(firmwareVersion)
	case strings.Contains(data, `"media.status"`):
		return sgdReply(r.mediaStatus())
	default:
		return sgdReply("?")
	}
}

func (r *SGDResponder) mediaStatus() string {
	if r.state.BlockingFault() == "paper_out" {
		return "out"
	}
	return "ok"
}

func sgdReply(value string) string {
	if value == "?" {
		return "?\r\n"
	}
	return fmt.Sprintf("\"%s\"\r\n", value)
}
