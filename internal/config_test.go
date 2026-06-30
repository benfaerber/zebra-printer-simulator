package internal

import (
	"testing"
	"time"
)

func TestLoadConfig_Defaults(t *testing.T) {
	clearEnv(t)
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.TCPPort != 19100 {
		t.Errorf("expected TCP port 19100, got %d", cfg.TCPPort)
	}
	if cfg.HTTPPort != 8081 {
		t.Errorf("expected HTTP port 8081, got %d", cfg.HTTPPort)
	}
	if cfg.Dpmm != 8 {
		t.Errorf("expected default dpmm 8 (203 dpi), got %d", cfg.Dpmm)
	}
	if cfg.BasicAuthEnabled() {
		t.Error("expected basic auth disabled by default")
	}
	if cfg.PrintDelay != 0 {
		t.Errorf("expected zero print delay, got %v", cfg.PrintDelay)
	}
}

func TestLoadConfig_DPI(t *testing.T) {
	tests := map[string]int{"203": 8, "300": 12, "600": 24}
	for dpi, want := range tests {
		clearEnv(t)
		t.Setenv("DPI", dpi)
		cfg, err := LoadConfig()
		if err != nil {
			t.Fatalf("dpi %s: %v", dpi, err)
		}
		if cfg.Dpmm != want {
			t.Errorf("dpi %s: expected dpmm %d, got %d", dpi, want, cfg.Dpmm)
		}
	}
}

func TestLoadConfig_InvalidDPI(t *testing.T) {
	clearEnv(t)
	t.Setenv("DPI", "400")
	if _, err := LoadConfig(); err == nil {
		t.Error("expected error for invalid DPI")
	}
}

func TestLoadConfig_BasicAuthRequiresBoth(t *testing.T) {
	clearEnv(t)
	t.Setenv("BASIC_AUTH_USER", "alice")
	if _, err := LoadConfig(); err == nil {
		t.Error("expected error when only user is set")
	}
}

func TestLoadConfig_PrintDelay(t *testing.T) {
	clearEnv(t)
	t.Setenv("PRINT_DELAY_MS", "250")
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PrintDelay != 250*time.Millisecond {
		t.Errorf("expected 250ms delay, got %v", cfg.PrintDelay)
	}
}

func TestLoadConfig_NegativeValuesRejected(t *testing.T) {
	clearEnv(t)
	t.Setenv("PRINT_DELAY_MS", "-1")
	if _, err := LoadConfig(); err == nil {
		t.Error("expected error for negative print delay")
	}
}

func clearEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"TCP_HOST", "TCP_PORT", "HTTP_HOST", "HTTP_PORT",
		"OUTPUT_DIR", "LABEL_SIZE", "DPI",
		"BASIC_AUTH_USER", "BASIC_AUTH_PASS",
		"PRINT_DELAY_MS", "MAX_OUTPUT_FILES", "WEBHOOK_URL",
	} {
		t.Setenv(k, "")
	}
}
