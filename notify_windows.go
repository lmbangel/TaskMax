//go:build windows

package main

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/go-toast/toast"
	"golang.org/x/sys/windows/registry"
)

// Toasts scale the image down from 512px, which stays sharp — the .ico used
// for the tray renders blurry here because Windows upscales a small frame.
//
//go:embed build/appicon.png
var toastIcon []byte

// toastIconPath is where the app icon was materialised on disk for toasts;
// set by setupToastApp during startup.
var toastIconPath string

// pushAgentToast shows a Windows toast whose click opens taskmax://task/<id>.
// Failures are logged and swallowed — a lost notification must never break
// the action that triggered it.
func pushAgentToast(title, body string, taskID uint) {
	go func() {
		n := toast.Notification{
			AppID:               "TaskMax",
			Title:               title,
			Message:             body,
			Icon:                toastIconPath,
			ActivationArguments: fmt.Sprintf("%s%d", taskURLPrefix, taskID),
			// ActivationType defaults to "protocol", which is what routes the
			// click through the taskmax:// handler registered at startup.
		}
		if err := n.Push(); err != nil {
			log.Printf("toast failed: %v", err)
		}
	}()
}

// setupToastApp registers "TaskMax" as an AppUserModelID so toasts carry the
// app's name and icon in their header — without this, Windows renders them
// with no attribution. The embedded icon is written into the data directory
// so the registry has a stable file path to point at.
func setupToastApp(dataDir string) {
	iconPath := filepath.Join(dataDir, "taskmax.png")
	if err := os.WriteFile(iconPath, toastIcon, 0o644); err != nil {
		log.Printf("toast icon write failed: %v", err)
	} else {
		toastIconPath = iconPath
	}
	// Remove the low-res .ico an earlier build materialised.
	_ = os.Remove(filepath.Join(dataDir, "taskmax.ico"))

	k, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Classes\AppUserModelId\TaskMax`, registry.ALL_ACCESS)
	if err != nil {
		log.Printf("toast app registration failed: %v", err)
		return
	}
	defer k.Close()
	_ = k.SetStringValue("DisplayName", "TaskMax")
	if toastIconPath != "" {
		_ = k.SetStringValue("IconUri", toastIconPath)
	}
}

// registerURLProtocol claims the taskmax:// scheme for this executable in
// HKCU (no elevation needed). Re-run on every startup so the handler always
// points at whichever copy of the app ran last.
func registerURLProtocol() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	root, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Classes\taskmax`, registry.ALL_ACCESS)
	if err != nil {
		log.Printf("protocol registration failed: %v", err)
		return
	}
	defer root.Close()
	_ = root.SetStringValue("", "URL:TaskMax Protocol")
	_ = root.SetStringValue("URL Protocol", "")

	cmd, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Classes\taskmax\shell\open\command`, registry.ALL_ACCESS)
	if err != nil {
		return
	}
	defer cmd.Close()
	_ = cmd.SetStringValue("", fmt.Sprintf(`"%s" "%%1"`, exe))
}
