//go:build !windows

package main

import "github.com/gen2brain/beeep"

// pushAgentToast on macOS/Linux is a plain notification: beeep has no
// click-through activation there, so the toast informs without navigating.
func pushAgentToast(title, body string, _ uint) {
	go func() {
		_ = beeep.Notify(title, body, "")
	}()
}

// registerURLProtocol is Windows-only; toast clicks elsewhere don't navigate.
func registerURLProtocol() {}
