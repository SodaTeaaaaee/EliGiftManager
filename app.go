package main

import (
	"context"
	"runtime"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/config"
)

// App wires Wails lifecycle hooks to the application configuration.
type App struct {
	ctx context.Context
	cfg config.App
}

// BootstrapPayload is the initial metadata returned to the frontend shell.
type BootstrapPayload struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Module      string   `json:"module"`
	Description string   `json:"description"`
	Runtime     string   `json:"runtime"`
	Frontend    string   `json:"frontend"`
	Highlights  []string `json:"highlights"`
}

func NewApp(cfg config.App) *App {
	return &App{cfg: cfg}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// Bootstrap returns the frontend metadata needed for the initial dashboard.
func (a *App) Bootstrap() BootstrapPayload {
	return BootstrapPayload{
		Name:        a.cfg.Name,
		Version:     a.cfg.Version,
		Module:      a.cfg.Module,
		Description: a.cfg.Description,
		Runtime:     runtime.Version(),
		Frontend:    a.cfg.FrontendRuntime,
		Highlights: []string{
			"Go backend uses internal packages for app configuration.",
			"Vue 3 single-file components are compiled through Vite.",
			"Deno installs npm dependencies and runs frontend tasks without a local Node.js installation.",
			"Wails remains the desktop shell, binding layer, and packaging toolchain.",
		},
	}
}
