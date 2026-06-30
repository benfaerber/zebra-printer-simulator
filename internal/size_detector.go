package internal

import (
	"regexp"
	"strconv"
)

var (
	pwPattern = regexp.MustCompile(`\^PW(\d+)`)
	llPattern = regexp.MustCompile(`\^LL(\d+)`)
)

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
