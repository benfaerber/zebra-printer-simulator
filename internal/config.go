package internal

import (
	"fmt"
	"math"
	"os"
	"strconv"
)

type Config struct {
	TCPHost        string
	TCPPort        int
	HTTPHost       string
	HTTPPort       int
	OutputDir      string
	LabelSize      LabelSize
	Dpmm           int
	BasicAuthUser  string
	BasicAuthPass  string
	MaxOutputFiles int
	WebhookURL     string
}

func LoadConfig() (Config, error) {
	dpmm, err := parseDpmm(envOrDefault("DPI", "203"))
	if err != nil {
		return Config{}, err
	}

	maxFiles, err := envIntOrDefault("MAX_OUTPUT_FILES", 0)
	if err != nil {
		return Config{}, fmt.Errorf("MAX_OUTPUT_FILES: %w", err)
	}
	if maxFiles < 0 {
		return Config{}, fmt.Errorf("MAX_OUTPUT_FILES must be >= 0, got %d", maxFiles)
	}

	tcpPort, err := envIntOrDefault("TCP_PORT", 19100)
	if err != nil {
		return Config{}, fmt.Errorf("TCP_PORT: %w", err)
	}
	httpPort, err := envIntOrDefault("HTTP_PORT", 8081)
	if err != nil {
		return Config{}, fmt.Errorf("HTTP_PORT: %w", err)
	}

	user := os.Getenv("BASIC_AUTH_USER")
	pass := os.Getenv("BASIC_AUTH_PASS")
	if (user == "") != (pass == "") {
		return Config{}, fmt.Errorf("BASIC_AUTH_USER and BASIC_AUTH_PASS must be set together")
	}

	return Config{
		TCPHost:        os.Getenv("TCP_HOST"),
		TCPPort:        tcpPort,
		HTTPHost:       os.Getenv("HTTP_HOST"),
		HTTPPort:       httpPort,
		OutputDir:      envOrDefault("OUTPUT_DIR", "./output"),
		LabelSize:      parseLabelSize(envOrDefault("LABEL_SIZE", "4x6")),
		Dpmm:           dpmm,
		BasicAuthUser:  user,
		BasicAuthPass:  pass,
		MaxOutputFiles: maxFiles,
		WebhookURL:     os.Getenv("WEBHOOK_URL"),
	}, nil
}

func (c Config) TCPAddr() string  { return fmt.Sprintf("%s:%d", c.TCPHost, c.TCPPort) }
func (c Config) HTTPAddr() string { return fmt.Sprintf("%s:%d", c.HTTPHost, c.HTTPPort) }

// HTTPURL is a clickable address for the dashboard. An empty bind host means
// "all interfaces", so localhost is the reachable host to surface in logs.
func (c Config) HTTPURL() string {
	host := c.HTTPHost
	if host == "" {
		host = "localhost"
	}
	return fmt.Sprintf("http://%s:%d", host, c.HTTPPort)
}
func (c Config) BasicAuthEnabled() bool {
	return c.BasicAuthUser != "" && c.BasicAuthPass != ""
}

// DPI reports the dots-per-inch matching the configured dot density, for SGD
// queries that report resolution in DPI rather than dpmm.
func (c Config) DPI() int {
	switch c.Dpmm {
	case 8:
		return 203
	case 12:
		return 300
	case 24:
		return 600
	default:
		return int(math.Round(float64(c.Dpmm) * 25.4))
	}
}

// PrintWidthDots is the label width expressed in dots, used for the SGD
// ezpl.print_width query.
func (c Config) PrintWidthDots() int {
	return int(math.Round(c.LabelSize.WidthMm / 25.4 * float64(c.DPI())))
}

func parseDpmm(dpi string) (int, error) {
	switch dpi {
	case "203":
		return 8, nil
	case "300":
		return 12, nil
	case "600":
		return 24, nil
	default:
		return 0, fmt.Errorf("DPI must be 203, 300, or 600, got %q", dpi)
	}
}

func parseLabelSize(s string) LabelSize {
	switch s {
	case "6x4":
		return LabelSize6x4
	case "2x4":
		return LabelSize2x4
	default:
		return LabelSize4x6
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envIntOrDefault(key string, fallback int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("not an integer: %q", v)
	}
	return n, nil
}
