package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/service"
)

const (
	zoomMinPercent = 25
	zoomMaxPercent = 500
)

func zoomFilePath() (string, error) {
	dataDir, err := service.ResolveDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataDir, "zoom.cfg"), nil
}

// LoadZoomPercent reads the saved zoom percentage from zoom.cfg. Returns 100
// if the file is missing, unreadable, unparsable (including NaN), or outside
// the 25–500 range. Values written by the legacy v2 binding ("125.00") still
// parse.
func LoadZoomPercent() float64 {
	path, err := zoomFilePath()
	if err != nil {
		return 100
	}
	return loadZoomPercentFrom(path)
}

// loadZoomPercentFrom is the path-injectable core of LoadZoomPercent. NaN must
// be rejected explicitly: all comparisons against NaN are false, so the range
// checks alone would let it through. ±Inf is already caught by them.
func loadZoomPercentFrom(path string) float64 {
	data, err := os.ReadFile(path)
	if err != nil {
		return 100
	}
	v, err := strconv.ParseFloat(string(data), 64)
	if err != nil || math.IsNaN(v) || v < zoomMinPercent || v > zoomMaxPercent {
		return 100
	}
	return v
}

// SaveZoomPercent clamps the zoom percentage to 25–500 and writes it to
// zoom.cfg as an integer percentage.
func SaveZoomPercent(percent float64) error {
	cfgPath, err := zoomFilePath()
	if err != nil {
		return err
	}
	return saveZoomPercentTo(cfgPath, percent)
}

// saveZoomPercentTo is the path-injectable core of SaveZoomPercent.
func saveZoomPercentTo(path string, percent float64) error {
	if percent < zoomMinPercent {
		percent = zoomMinPercent
	}
	if percent > zoomMaxPercent {
		percent = zoomMaxPercent
	}
	return os.WriteFile(path, fmt.Appendf(nil, "%d", int64(percent+0.5)), 0o644)
}
