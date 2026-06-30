package internal

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestOutputRetention_NoCapKeepsAll(t *testing.T) {
	dir := t.TempDir()
	createPNGs(t, dir, 5)

	NewOutputRetention(dir, 0).Enforce()

	if got := countPNGs(t, dir); got != 5 {
		t.Errorf("expected 5 files kept (no cap), got %d", got)
	}
}

func TestOutputRetention_KeepsNewest(t *testing.T) {
	dir := t.TempDir()
	paths := createPNGs(t, dir, 5)

	NewOutputRetention(dir, 2).Enforce()

	if got := countPNGs(t, dir); got != 2 {
		t.Errorf("expected 2 files kept, got %d", got)
	}
	for _, p := range paths[:3] {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("expected oldest file %s to be deleted", filepath.Base(p))
		}
	}
	for _, p := range paths[3:] {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected newest file %s to remain", filepath.Base(p))
		}
	}
}

func TestOutputRetention_IgnoresNonPNG(t *testing.T) {
	dir := t.TempDir()
	createPNGs(t, dir, 3)
	other := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(other, []byte("keep me"), 0644); err != nil {
		t.Fatal(err)
	}

	NewOutputRetention(dir, 1).Enforce()

	if _, err := os.Stat(other); err != nil {
		t.Error("non-PNG file should be untouched by retention")
	}
}

func createPNGs(t *testing.T, dir string, n int) []string {
	t.Helper()
	paths := make([]string, n)
	for i := 0; i < n; i++ {
		p := filepath.Join(dir, "label_"+timeSuffix(i)+".png")
		if err := os.WriteFile(p, []byte("png"), 0644); err != nil {
			t.Fatal(err)
		}
		past := time.Now().Add(time.Duration(i-n) * time.Second)
		if err := os.Chtimes(p, past, past); err != nil {
			t.Fatal(err)
		}
		paths[i] = p
	}
	return paths
}

func countPNGs(t *testing.T, dir string) int {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".png" {
			n++
		}
	}
	return n
}

func timeSuffix(i int) string {
	return time.Date(2026, 1, 1, 0, 0, i, 0, time.UTC).Format("20060102_150405")
}
