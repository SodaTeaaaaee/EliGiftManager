package main

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/service"
)

func zoomFilePath() (string, error) {
	dataDir, err := service.ResolveDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataDir, "zoom.cfg"), nil
}

// LoadZoom reads the saved zoom percentage from zoom.cfg. Returns 100 if no saved value.
func LoadZoom() float64 {
	path, err := zoomFilePath()
	if err != nil {
		return 100
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return 100
	}
	v, err := strconv.ParseFloat(string(data), 64)
	if err != nil || v < 25 || v > 500 {
		return 100
	}
	return v
}
