package internal

import (
	"log/slog"
	"os"
	"path/filepath"
	"sort"
)

type OutputRetention struct {
	dir   string
	keep  int
}

func NewOutputRetention(dir string, keep int) *OutputRetention {
	return &OutputRetention{dir: dir, keep: keep}
}

func (r *OutputRetention) Enforce() {
	if r.keep <= 0 {
		return
	}

	pngs, err := r.collectPNGs()
	if err != nil {
		slog.Warn("retention scan failed", "err", err)
		return
	}
	if len(pngs) <= r.keep {
		return
	}

	r.deleteOldest(pngs, len(pngs)-r.keep)
}

type pngEntry struct {
	path    string
	modTime int64
}

func (r *OutputRetention) collectPNGs() ([]pngEntry, error) {
	entries, err := os.ReadDir(r.dir)
	if err != nil {
		return nil, err
	}

	var pngs []pngEntry
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".png" {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		pngs = append(pngs, pngEntry{
			path:    filepath.Join(r.dir, e.Name()),
			modTime: info.ModTime().UnixNano(),
		})
	}

	sort.Slice(pngs, func(i, j int) bool {
		return pngs[i].modTime < pngs[j].modTime
	})
	return pngs, nil
}

func (r *OutputRetention) deleteOldest(pngs []pngEntry, count int) {
	for i := 0; i < count && i < len(pngs); i++ {
		if err := os.Remove(pngs[i].path); err != nil {
			slog.Warn("retention delete failed", "path", pngs[i].path, "err", err)
		}
	}
}
