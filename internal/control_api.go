package internal

import (
	"bytes"
	"crypto/subtle"
	_ "embed"
	"encoding/json"
	"fmt"
	"image/png"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type ControlAPI struct {
	state     *PrinterState
	renderer  *Renderer
	outputDir string
	authUser  string
	authPass  string
}

type ControlAPIOptions struct {
	State         *PrinterState
	Renderer      *Renderer
	OutputDir     string
	BasicAuthUser string
	BasicAuthPass string
}

func NewControlAPI(opts ControlAPIOptions) *ControlAPI {
	return &ControlAPI{
		state:     opts.State,
		renderer:  opts.Renderer,
		outputDir: opts.OutputDir,
		authUser:  opts.BasicAuthUser,
		authPass:  opts.BasicAuthPass,
	}
}

//go:embed dashboard.html
var dashboardHTML []byte

//go:embed preview.html
var previewHTML []byte

//go:embed favicon.svg
var faviconSVG []byte

const previewMaxBytes = 1 << 20 // 1 MiB

func (a *ControlAPI) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", a.getHealthz)
	mux.HandleFunc("GET /favicon.svg", a.getFavicon)
	mux.HandleFunc("GET /favicon.ico", a.getFavicon)
	mux.HandleFunc("GET /status", a.protect(a.getStatus))
	mux.HandleFunc("POST /config", a.protect(a.postConfig))
	mux.HandleFunc("POST /reset", a.protect(a.postReset))
	mux.HandleFunc("GET /jobs", a.protect(a.getJobs))
	mux.HandleFunc("DELETE /jobs/{filename}", a.protect(a.deleteJob))
	mux.HandleFunc("GET /metrics", a.protect(a.getMetrics))
	mux.HandleFunc("GET /preview", a.protect(a.getPreview))
	mux.HandleFunc("POST /preview", a.protect(a.postPreview))
	mux.HandleFunc("GET /", a.protect(a.getDashboard))
	mux.Handle("GET /images/", a.protectHandler(http.StripPrefix("/images/",
		http.FileServer(http.Dir(a.outputDir)))))
	return mux
}

func (a *ControlAPI) basicAuthEnabled() bool {
	return a.authUser != "" && a.authPass != ""
}

func (a *ControlAPI) protect(next http.HandlerFunc) http.HandlerFunc {
	if !a.basicAuthEnabled() {
		return next
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if !a.checkAuth(r) {
			a.requireAuth(w)
			return
		}
		next(w, r)
	}
}

func (a *ControlAPI) protectHandler(next http.Handler) http.Handler {
	if !a.basicAuthEnabled() {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.checkAuth(r) {
			a.requireAuth(w)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *ControlAPI) checkAuth(r *http.Request) bool {
	user, pass, ok := r.BasicAuth()
	if !ok {
		return false
	}
	userMatch := subtle.ConstantTimeCompare([]byte(user), []byte(a.authUser)) == 1
	passMatch := subtle.ConstantTimeCompare([]byte(pass), []byte(a.authPass)) == 1
	return userMatch && passMatch
}

func (a *ControlAPI) requireAuth(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="printer-simulator"`)
	http.Error(w, "unauthorized", http.StatusUnauthorized)
}

func (a *ControlAPI) getHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (a *ControlAPI) getFavicon(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = w.Write(faviconSVG)
}

func (a *ControlAPI) getStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, a.state.Snapshot())
}

type configRequest struct {
	Flag    string `json:"flag"`
	Enabled bool   `json:"enabled"`
}

func (a *ControlAPI) postConfig(w http.ResponseWriter, r *http.Request) {
	var req configRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Flag == "sgd" {
		a.state.SetSGDEnabled(req.Enabled)
	} else {
		a.state.SetError(req.Flag, req.Enabled)
	}

	slog.Info("config updated", "flag", req.Flag, "enabled", req.Enabled)
	writeJSON(w, a.state.Snapshot())
}

func (a *ControlAPI) postReset(w http.ResponseWriter, r *http.Request) {
	a.state.Reset()
	deleted := a.clearOutputDir()
	slog.Info("simulator reset", "deleted_files", deleted)
	writeJSON(w, map[string]interface{}{
		"status":        "reset",
		"deleted_files": deleted,
	})
}

func (a *ControlAPI) clearOutputDir() int {
	entries, err := os.ReadDir(a.outputDir)
	if err != nil {
		return 0
	}
	deleted := 0
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".png" {
			continue
		}
		if err := os.Remove(filepath.Join(a.outputDir, e.Name())); err != nil {
			slog.Warn("reset: delete failed", "name", e.Name(), "err", err)
			continue
		}
		deleted++
	}
	return deleted
}

type labelInfo struct {
	Filename  string `json:"filename"`
	URL       string `json:"url"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	SizeBytes int64  `json:"size_bytes"`
	Timestamp string `json:"timestamp"`
}

func (a *ControlAPI) getJobs(w http.ResponseWriter, r *http.Request) {
	entries, err := os.ReadDir(a.outputDir)
	if err != nil {
		writeJSON(w, map[string]interface{}{"labels": []labelInfo{}, "count": 0})
		return
	}

	var labels []labelInfo
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".png" {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		li := labelInfo{
			Filename:  e.Name(),
			URL:       fmt.Sprintf("/images/%s", e.Name()),
			SizeBytes: info.Size(),
			Timestamp: info.ModTime().Format("2006-01-02 15:04:05"),
		}
		if w, h := readPNGDimensions(filepath.Join(a.outputDir, e.Name())); w > 0 {
			li.Width = w
			li.Height = h
		}
		labels = append(labels, li)
	}

	sort.Slice(labels, func(i, j int) bool {
		return labels[i].Timestamp > labels[j].Timestamp
	})

	writeJSON(w, map[string]interface{}{
		"labels": labels,
		"count":  len(labels),
	})
}

func (a *ControlAPI) deleteJob(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("filename")
	if !isSafeLabelName(name) {
		http.Error(w, "invalid filename", http.StatusBadRequest)
		return
	}
	path := filepath.Join(a.outputDir, name)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		slog.Warn("delete failed", "name", name, "err", err)
		http.Error(w, "delete failed", http.StatusInternalServerError)
		return
	}
	slog.Info("label deleted", "name", name)
	writeJSON(w, map[string]interface{}{"status": "deleted", "filename": name})
}

func isSafeLabelName(name string) bool {
	if name == "" || filepath.Ext(name) != ".png" {
		return false
	}
	if strings.ContainsAny(name, `/\`) || strings.Contains(name, "..") {
		return false
	}
	if filepath.Base(name) != name {
		return false
	}
	return true
}

func (a *ControlAPI) getMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	_, _ = w.Write([]byte(renderPrometheusMetrics(a.state.MetricsSnapshot())))
}

func renderPrometheusMetrics(m MetricsSnapshot) string {
	var b strings.Builder

	writeCounter(&b, "printer_labels_total",
		"Total number of labels successfully rendered.", m.LabelCount)
	writeGauge(&b, "printer_formats_in_buffer",
		"Number of label formats currently being rendered.", m.FormatsInBuffer)
	writeCounter(&b, "printer_render_failures_total",
		"Total number of label render failures.", m.RenderFailures)
	writeGauge(&b, "printer_sgd_enabled",
		"1 if SGD responses are enabled, 0 if disabled.", boolToInt(m.SGDEnabled))

	b.WriteString("# HELP printer_fault Fault flag status (1 = active).\n")
	b.WriteString("# TYPE printer_fault gauge\n")
	flags := []string{"paper_out", "paused", "head_up", "ribbon_out", "under_temp", "over_temp"}
	for _, f := range flags {
		fmt.Fprintf(&b, "printer_fault{flag=%q} %d\n", f, boolToInt(m.Faults[f]))
	}

	return b.String()
}

func writeCounter(b *strings.Builder, name, help string, value int) {
	fmt.Fprintf(b, "# HELP %s %s\n# TYPE %s counter\n%s %d\n", name, help, name, name, value)
}

func writeGauge(b *strings.Builder, name, help string, value int) {
	fmt.Fprintf(b, "# HELP %s %s\n# TYPE %s gauge\n%s %d\n", name, help, name, name, value)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (a *ControlAPI) getDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(dashboardHTML)
}

func (a *ControlAPI) getPreview(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(previewHTML)
}

func (a *ControlAPI) postPreview(w http.ResponseWriter, r *http.Request) {
	if a.renderer == nil {
		http.Error(w, "preview unavailable", http.StatusServiceUnavailable)
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, previewMaxBytes))
	if err != nil {
		http.Error(w, "request body too large or unreadable", http.StatusBadRequest)
		return
	}
	if len(body) == 0 {
		http.Error(w, "empty ZPL body", http.StatusBadRequest)
		return
	}
	var buf bytes.Buffer
	if err := a.renderer.RenderPreview(body, &buf); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(buf.Bytes())
}

func readPNGDimensions(path string) (int, int) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0
	}
	defer f.Close()
	cfg, err := png.DecodeConfig(f)
	if err != nil {
		return 0, 0
	}
	return cfg.Width, cfg.Height
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
