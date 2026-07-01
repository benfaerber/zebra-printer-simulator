package internal

import (
	"fmt"
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
func (c Config) BasicAuthEnabled() bool {
	return c.BasicAuthUser != "" && c.BasicAuthPass != ""
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
