package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"path/filepath"
	"time"
)

type LabelPrintedEvent struct {
	Filename   string `json:"filename"`
	Path       string `json:"path"`
	LabelCount int    `json:"label_count"`
}

type Webhook interface {
	Notify(event LabelPrintedEvent)
}

type NoopWebhook struct{}

func (NoopWebhook) Notify(LabelPrintedEvent) {}

type HTTPWebhook struct {
	url    string
	client *http.Client
}

func NewHTTPWebhook(url string) *HTTPWebhook {
	return &HTTPWebhook{
		url:    url,
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

func (w *HTTPWebhook) Notify(event LabelPrintedEvent) {
	go w.send(event)
}

func (w *HTTPWebhook) send(event LabelPrintedEvent) {
	body, err := json.Marshal(event)
	if err != nil {
		slog.Warn("webhook marshal failed", "err", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.url, bytes.NewReader(body))
	if err != nil {
		slog.Warn("webhook request failed", "err", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		slog.Warn("webhook POST failed", "url", w.url, "err", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		slog.Warn("webhook returned error status",
			"url", w.url, "status", resp.StatusCode, "filename", event.Filename)
	}
}

func NewWebhook(url string) Webhook {
	if url == "" {
		return NoopWebhook{}
	}
	return NewHTTPWebhook(url)
}

func eventFromPath(path string, labelCount int) LabelPrintedEvent {
	return LabelPrintedEvent{
		Filename:   filepath.Base(path),
		Path:       path,
		LabelCount: labelCount,
	}
}
