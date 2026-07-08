package main

import (
	"embed"
	"log"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"taskmax/internal/config"
	"taskmax/internal/db"
)

//go:embed all:frontend/dist
var assets embed.FS

// dataDir decides where config.yaml and the sqlite database live.
//
// Portable mode: if a config.yaml already sits in the working directory or
// next to the executable (dev checkouts, portable exe, USB stick), keep
// using that directory. Otherwise fall back to the per-user config dir
// (%AppData%\TaskMax on Windows) — an installed copy under Program Files
// must never write beside its own binary.
func dataDir() string {
	if _, err := os.Stat("config.yaml"); err == nil {
		return "."
	}
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		if _, err := os.Stat(filepath.Join(exeDir, "config.yaml")); err == nil {
			return exeDir
		}
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "."
	}
	dir := filepath.Join(base, "TaskMax")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "."
	}
	return dir
}

func main() {
	dir := dataDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Resolve a relative sqlite path inside the data dir without touching the
	// config we bind and save — the file keeps its portable relative DSN.
	dbCfg := *cfg
	if dbCfg.Database.Type == "sqlite" && !filepath.IsAbs(dbCfg.Database.DSN) {
		dbCfg.Database.DSN = filepath.Join(dir, dbCfg.Database.DSN)
	}

	database, err := db.NewDB(&dbCfg)
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
