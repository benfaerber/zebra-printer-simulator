package internal

import "time"

// PrintSpeed models how fast the simulated printer renders a label. Slower
// speeds add more artificial delay per job, mimicking the media feed rate of a
// physical printer. It is selected at runtime from the dashboard rather than
// fixed by an environment variable.
type PrintSpeed string

const (
	PrintSpeedFast   PrintSpeed = "fast"
	PrintSpeedNormal PrintSpeed = "normal"
	PrintSpeedSlow   PrintSpeed = "slow"
)

// DefaultPrintSpeed is the speed a freshly started or reset printer uses.
const DefaultPrintSpeed = PrintSpeedFast

var printSpeedDelays = map[PrintSpeed]time.Duration{
	PrintSpeedFast:   0,
	PrintSpeedNormal: 500 * time.Millisecond,
	PrintSpeedSlow:   2 * time.Second,
}

// ParsePrintSpeed converts a wire value into a PrintSpeed, reporting whether it
// names a known speed.
func ParsePrintSpeed(s string) (PrintSpeed, bool) {
	speed := PrintSpeed(s)
	_, ok := printSpeedDelays[speed]
	return speed, ok
}

// Delay is the artificial per-label render delay for this speed.
func (p PrintSpeed) Delay() time.Duration {
	return printSpeedDelays[p]
}

// Valid reports whether this speed is one of the known selectable options.
func (p PrintSpeed) Valid() bool {
	_, ok := printSpeedDelays[p]
	return ok
}
