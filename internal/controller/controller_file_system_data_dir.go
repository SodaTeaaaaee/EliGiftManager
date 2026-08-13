package controller

// GetDataDir exposes the resolved data directory path to the frontend.
func (c *FileSystemController) GetDataDir() (string, error) {
	return c.resolvedDataDir()
}
