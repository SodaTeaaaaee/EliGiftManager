package config

// App defines the static application metadata used by Wails and the frontend.
type App struct {
	Name            string
	Version         string
	Module          string
	Description     string
	FrontendRuntime string
	WindowWidth     int
	WindowHeight    int
	MinWindowWidth  int
	MinWindowHeight int
}

func Load() App {
	return App{
		Name:            "EliGiftManager",
		Version:         "0.1.0",
		Module:          "github.com/SodaTeaaaaee/EliGiftManager",
		Description:     "A desktop gift planning workspace built with Go, Wails, Vue 3 SFC, Vite, and Deno.",
		FrontendRuntime: "Vue 3.5.33 + Vite 8.0.9 + Deno 2",
		WindowWidth:     1320,
		WindowHeight:    860,
		MinWindowWidth:  1080,
		MinWindowHeight: 720,
	}
}
