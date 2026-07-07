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

	err = wails.Run(&options.App{
		Title:  "TaskMax",
		Width:  1200,
		Height: 800,
		MinWidth:  900,
		MinHeight: 640,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour:  &options.RGBA{R: 30, G: 30, B: 46, A: 1},
		HideWindowOnClose: cfg.App.MinimizeToTray,
		OnStartup:         app.startup,
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		log.Fatalf("failed to run app: %v", err)
	}
}
