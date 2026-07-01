package internal

import (
	"regexp"
	"strconv"
)

var (
	pwPattern = regexp.MustCompile(`\^PW(\d+)`)
	llPattern = regexp.MustCompile(`\^LL(\d+)`)
	pqPattern = regexp.MustCompile(`\^PQ(\d+)`)
)

// maxPrintQuantity caps the copies honored from a single ^PQ command so a
// malformed or hostile job cannot flood the output directory.
const maxPrintQuantity = 100

// detectPrintQuantity reads the copy count from a ^PQ command, defaulting to a
// single copy and clamping to maxPrintQuantity.
func detectPrintQuantity(data []byte) int {
	q := extractFirstInt(pqPattern, string(data))
	if q < 1 {
		return 1
	}
	if q > maxPrintQuantity {
		return maxPrintQuantity
	}
	return q
}

func detectLabelSize(data []byte, fallback LabelSize, dpmm int) LabelSize {
	if dpmm <= 0 {
		return fallback
	}

	zpl := string(data)

	widthDots := extractFirstInt(pwPattern, zpl)
	heightDots := extractFirstInt(llPattern, zpl)

	if widthDots == 0 && heightDots == 0 {
		return fallback
	}

	result := fallback
	if widthDots > 0 {
		result.WidthMm = float64(widthDots) / float64(dpmm)
	}
	if heightDots > 0 {
		result.HeightMm = float64(heightDots) / float64(dpmm)
	}
	return result
}

func extractFirstInt(re *regexp.Regexp, s string) int {
	match := re.FindStringSubmatch(s)
	if len(match) < 2 {
		return 0
	}
	v, err := strconv.Atoi(match[1])
	if err != nil {
		return 0
	}
	return v
}
