//go:build !windows

package main

// Launch-on-startup is only wired up on Windows for now; other platforms
// report it as unavailable.

func launchOnStartupEnabled() (bool, error) {
	return false, nil
}

func setLaunchOnStartup(enabled bool) error {
	return nil
}
