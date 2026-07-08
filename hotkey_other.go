//go:build !windows

package main

// The global hotkey is only wired up on Windows for now. macOS/Linux need
// cgo-backed listeners (golang.design/x/hotkey) — revisit when those builds
// get real usage.
func (a *App) registerHotkey() {}
