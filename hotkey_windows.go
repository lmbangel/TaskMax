//go:build windows

package main

import (
	"log"

	"golang.design/x/hotkey"
)

// registerHotkey binds Ctrl+Alt+D to toggling the widget's visibility.
// Registration failures (e.g. another app owns the combination) are logged
// and otherwise ignored — the app works fine without the hotkey.
func (a *App) registerHotkey() {
	hk := hotkey.New([]hotkey.Modifier{hotkey.ModCtrl, hotkey.ModAlt}, hotkey.KeyD)
	if err := hk.Register(); err != nil {
		log.Printf("global hotkey unavailable: %v", err)
		return
	}
	go func() {
		for range hk.Keydown() {
			a.toggleVisibility()
		}
	}()
}
