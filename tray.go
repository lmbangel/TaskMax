package main

import (
	_ "embed"

	"github.com/energye/systray"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed build/windows/icon.ico
var trayIcon []byte

// startTray launches the system tray icon on its own goroutine. It gives the
// widget a home while hidden: left-click restores the window, right-click
// opens a small menu. Without it, hiding on close would strand the app.
func (a *App) startTray() {
	go systray.Run(a.trayReady, nil)
}

func (a *App) trayReady() {
	systray.SetIcon(trayIcon)
	systray.SetTooltip("TaskMax")

	systray.SetOnClick(func(_ systray.IMenu) { a.showWindow() })
	systray.SetOnRClick(func(menu systray.IMenu) { _ = menu.ShowMenu() })

	show := systray.AddMenuItem("Show TaskMax", "Bring the widget back")
	show.Click(a.showWindow)
	systray.AddSeparator()
	quit := systray.AddMenuItem("Quit TaskMax", "Exit TaskMax")
	quit.Click(func() { wailsruntime.Quit(a.ctx) })
}

// showWindow restores the widget from the tray or the taskbar.
func (a *App) showWindow() {
	wailsruntime.WindowShow(a.ctx)
	wailsruntime.WindowUnminimise(a.ctx)
}

// stopTray removes the tray icon; called from the app's shutdown hook.
func (a *App) stopTray() {
	systray.Quit()
}
