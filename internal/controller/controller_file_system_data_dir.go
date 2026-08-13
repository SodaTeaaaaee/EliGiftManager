package controller

import (
	"github.com/SodaTeaaaaee/EliGiftManager/internal/service"
)

// GetDataDir exposes the resolved data directory path to the frontend.
func (c *FileSystemController) GetDataDir() (string, error) {
	return service.ResolveDataDir()
}
