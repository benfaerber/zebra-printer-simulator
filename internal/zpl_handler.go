package internal

import (
	"fmt"
	"strings"
)

type CommandType int

const (
	CommandUnknown CommandType = iota
	CommandHS
	CommandSGD
	CommandZPL
)

func ClassifyInput(data string) CommandType {
	trimmed := strings.TrimSpace(data)

	if strings.HasPrefix(trimmed, "~HS") {
		return CommandHS
	}

	if strings.HasPrefix(trimmed, "! U1 getvar") || strings.HasPrefix(trimmed, "! U1 setvar") {
		return CommandSGD
	}

	if strings.Contains(trimmed, "^XA") {
		return CommandZPL
	}

	return CommandUnknown
}

func HandleSGDCommand(data string, state *PrinterState) string {
	if !state.SGDEnabled() {
		return "?\r\n"
	}

	if strings.Contains(data, `"odometer.total_label_count"`) {
		return fmt.Sprintf("\"%d\"\r\n", state.LabelCount())
	}

	if strings.Contains(data, `"head.resolution.in_dpi"`) {
		return "\"203\"\r\n"
	}

	if strings.Contains(data, `"ezpl.print_width"`) {
		return "\"832\"\r\n"
	}

	return "?\r\n"
}
