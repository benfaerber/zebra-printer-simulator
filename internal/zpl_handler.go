package internal

import (
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
