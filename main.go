package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"taskmax/internal/config"
	"taskmax/internal/db"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	const cfgPath = "config.yaml"

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	database, err := db.NewDB(cfg)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	if err := db.Migrate(database); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	app := NewApp(cfg, cfgPath, database)

	// Compact "desk widget" window: frameless, always on top, calculator-sized.
	// It is positioned at the bottom-right of the screen on startup (see
	// App.startup) and dragged via the custom titlebar in the frontend.
	err = wails.Run(&options.App{
		Title:     "TaskMax",
		Width:     windowWidth,
		Height:    windowHeight,
		MinWidth:  340,
		MinHeight: 520,
		MaxWidth:  460,
		MaxHeight: 760,
		Frameless:   true,
		AlwaysOnTop: true,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour:  &options.RGBA{R: 28, G: 27, B: 25, A: 1},
		HideWindowOnClose: cfg.App.MinimizeToTray,
		OnStartup:         app.startup,
		OnShutdown:        app.shutdown,
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		log.Fatalf("failed to run app: %v", err)
	}
}
