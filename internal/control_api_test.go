package internal

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestControlAPI_SetPrintSpeed(t *testing.T) {
	state := NewPrinterState()
	api := NewControlAPI(ControlAPIOptions{State: state, OutputDir: t.TempDir()})

	req := httptest.NewRequest(http.MethodPost, "/config",
		strings.NewReader(`{"flag":"speed","speed":"slow"}`))
	rec := httptest.NewRecorder()
	api.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if state.PrintSpeed() != PrintSpeedSlow {
		t.Errorf("expected print speed slow, got %q", state.PrintSpeed())
	}
}

func TestControlAPI_RejectsInvalidPrintSpeed(t *testing.T) {
	state := NewPrinterState()
	api := NewControlAPI(ControlAPIOptions{State: state, OutputDir: t.TempDir()})

	req := httptest.NewRequest(http.MethodPost, "/config",
		strings.NewReader(`{"flag":"speed","speed":"turbo"}`))
	rec := httptest.NewRecorder()
	api.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid speed, got %d", rec.Code)
	}
	if state.PrintSpeed() != DefaultPrintSpeed {
		t.Errorf("expected speed unchanged on rejection, got %q", state.PrintSpeed())
	}
}

func TestControlAPI_NoAuthByDefault(t *testing.T) {
	api := NewControlAPI(ControlAPIOptions{
		State:     NewPrinterState(),
		OutputDir: t.TempDir(),
	})

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	rec := httptest.NewRecorder()
	api.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 without auth when disabled, got %d", rec.Code)
	}
}

func TestControlAPI_BasicAuthRequiredWhenConfigured(t *testing.T) {
	api := NewControlAPI(ControlAPIOptions{
		State:         NewPrinterState(),
		OutputDir:     t.TempDir(),
		BasicAuthUser: "alice",
		BasicAuthPass: "secret",
	})

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	rec := httptest.NewRecorder()
	api.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without credentials, got %d", rec.Code)
	}
	if got := rec.Header().Get("WWW-Authenticate"); !strings.Contains(got, "Basic") {
		t.Errorf("expected WWW-Authenticate header, got %q", got)
	}
}

func TestControlAPI_BasicAuthAcceptsValidCreds(t *testing.T) {
	api := NewControlAPI(ControlAPIOptions{
		State:         NewPrinterState(),
		OutputDir:     t.TempDir(),
		BasicAuthUser: "alice",
		BasicAuthPass: "secret",
	})

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("alice:secret")))
	rec := httptest.NewRecorder()
	api.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 with valid creds, got %d", rec.Code)
	}
}

func TestControlAPI_HealthzAlwaysOpen(t *testing.T) {
	api := NewControlAPI(ControlAPIOptions{
		State:         NewPrinterState(),
		OutputDir:     t.TempDir(),
		BasicAuthUser: "alice",
		BasicAuthPass: "secret",
	})

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	api.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected /healthz to bypass auth, got %d", rec.Code)
	}
}

func TestControlAPI_Reset(t *testing.T) {
	state := NewPrinterState()
	state.IncrementLabelCount()
	state.IncrementLabelCount()
	state.SetError("paper_out", true)

	api := NewControlAPI(ControlAPIOptions{
		State:     state,
		OutputDir: t.TempDir(),
	})

	req := httptest.NewRequest(http.MethodPost, "/reset", nil)
	rec := httptest.NewRecorder()
	api.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if state.LabelCount() != 0 {
		t.Errorf("expected label count reset to 0, got %d", state.LabelCount())
	}
	snap := state.Snapshot()
	if snap["paper_out"] != false {
		t.Error("expected paper_out reset to false")
	}
}

func TestControlAPI_MetricsFormat(t *testing.T) {
	state := NewPrinterState()
	state.IncrementLabelCount()
	state.SetError("paused", true)

	api := NewControlAPI(ControlAPIOptions{State: state, OutputDir: t.TempDir()})

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	api.Handler().ServeHTTP(rec, req)

	body := rec.Body.String()
	for _, want := range []string{
		"# TYPE printer_labels_total counter",
		"printer_labels_total 1",
		"# TYPE printer_fault gauge",
		`printer_fault{flag="paused"} 1`,
		`printer_fault{flag="paper_out"} 0`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("metrics body missing %q\n--- body ---\n%s", want, body)
		}
	}
}
