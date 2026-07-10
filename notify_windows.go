//go:build windows

package main

import (
	"fmt"
	"log"
	"os"

	"github.com/go-toast/toast"
	"golang.org/x/sys/windows/registry"
)

// pushAgentToast shows a Windows toast whose click opens taskmax://task/<id>.
// Failures are logged and swallowed — a lost notification must never break
// the action that triggered it.
func pushAgentToast(title, body string, taskID uint) {
	go func() {
		n := toast.Notification{
			AppID:               "TaskMax",
			Title:               title,
			Message:             body,
			ActivationArguments: fmt.Sprintf("%s%d", taskURLPrefix, taskID),
			// ActivationType defaults to "protocol", which is what routes the
			// click through the taskmax:// handler registered at startup.
		}
		if err := n.Push(); err != nil {
			log.Printf("toast failed: %v", err)
		}
	}()
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
