package main

import (
	"os"
	"path/filepath"
	"testing"
)

// The tests target the path-injectable cores (loadZoomPercentFrom /
// saveZoomPercentTo) instead of chdir-ing: under the production build tag
// ResolveDataDir ignores the working directory (exe dir / user config dir),
// so chdir-based tests would read and write the real user config. The
// exported wrappers only add path resolution on top of these cores.

func TestLoadZoomPercentFrom(t *testing.T) {
	tests := []struct {
		name    string
		content string // written to zoom.cfg
		missing bool   // omit the file entirely
		want    float64
	}{
		{name: "missing file", missing: true, want: 100},
		{name: "empty file", content: "", want: 100},
		{name: "garbage text", content: "not-a-number", want: 100},
		{name: "NaN passes range checks but must fall back", content: "NaN", want: 100},
		{name: "+Inf above range", content: "+Inf", want: 100},
		{name: "-Inf below range", content: "-Inf", want: 100},
		{name: "legacy v2 fractional format", content: "125.00", want: 125},
		{name: "lower bound", content: "25", want: 25},
		{name: "upper bound", content: "500", want: 500},
		{name: "below range falls back to default, not clamp", content: "10", want: 100},
		{name: "above range falls back to default, not clamp", content: "900", want: 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "zoom.cfg")
			if !tt.missing {
				if err := os.WriteFile(path, []byte(tt.content), 0o644); err != nil {
					t.Fatalf("write zoom.cfg: %v", err)
				}
			}
			if got := loadZoomPercentFrom(path); got != tt.want {
				t.Fatalf("loadZoomPercentFrom with content %q = %v, want %v", tt.content, got, tt.want)
			}
		})
	}
}

func TestSaveZoomPercentRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		save float64
		want float64 // value Load must report after Save
	}{
		{name: "in-range value round-trips", save: 133, want: 133},
		{name: "fractional value rounds to nearest integer", save: 133.7, want: 134},
		{name: "below range clamps to 25 on save", save: 10, want: 25},
		{name: "above range clamps to 500 on save", save: 900, want: 500},
		// Documents why the WindowClosing hook skips non-positive sentinels:
		// SaveZoomPercent itself would clamp them to 25 and poison zoom.cfg.
		{name: "GetZoom error sentinel would clamp to 25 on save", save: -100, want: 25},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "zoom.cfg")
			if err := saveZoomPercentTo(path, tt.save); err != nil {
				t.Fatalf("saveZoomPercentTo(%v): %v", tt.save, err)
			}
			if got := loadZoomPercentFrom(path); got != tt.want {
				t.Fatalf("after saveZoomPercentTo(%v), load = %v, want %v", tt.save, got, tt.want)
			}
		})
	}
}
