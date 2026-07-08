//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	assetWindowsInstaller = "TaskMax-windows-amd64-installer.exe"
	assetWindowsPortable  = "TaskMax-windows-amd64-portable.exe"
)

func selfUpdateSupported() bool { return true }

// applyUpdate picks the right strategy for how this copy was deployed:
//
//   - Installed (NSIS uninstaller sits next to the exe): download the new
//     installer and run it silently. The installer closes the running app,
//     writes into Program Files under its own UAC elevation, and the helper
//     relaunches TaskMax when it finishes.
//   - Portable: download the new exe and swap it in place, then relaunch.
func (a *App) applyUpdate(release *releaseInfo) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(exe), "uninstall.exe")); err == nil {
		return a.updateViaInstaller(release, exe)
	}
	return a.updatePortable(release, exe)
}

func (a *App) updateViaInstaller(release *releaseInfo, exe string) error {
	asset := release.asset(assetWindowsInstaller)
	if asset == nil {
		return fmt.Errorf("release %s has no Windows installer asset", release.TagName)
	}
	dir, err := updateTempDir()
	if err != nil {
		return err
	}
	installer := filepath.Join(dir, asset.Name)
	if err := a.downloadAsset(asset, installer); err != nil {
		return err
	}

	a.emitUpdateProgress("installing", 100)

	// Detached helper: wait for the silent install to finish (start /wait
	// rides through the UAC prompt), then bring TaskMax back. `&` chains
	// unconditionally, so a cancelled UAC prompt still relaunches the
	// current version instead of leaving the user with nothing.
	cmd := exec.Command("cmd", "/C",
		fmt.Sprintf(`start /wait "" "%s" /S & start "" "%s"`, installer, exe))
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x00000008} // DETACHED_PROCESS
	if err := cmd.Start(); err != nil {
		return err
	}

	a.emitUpdateProgress("restarting", 100)
	wailsruntime.Quit(a.ctx)
	return nil
}

func (a *App) updatePortable(release *releaseInfo, exe string) error {
	asset := release.asset(assetWindowsPortable)
	if asset == nil {
		return fmt.Errorf("release %s has no Windows portable asset", release.TagName)
	}
	// Download beside the running exe so the final rename is same-volume.
	newExe := exe + ".new"
	if err := a.downloadAsset(asset, newExe); err != nil {
		_ = os.Remove(newExe)
		return err
	}

	a.emitUpdateProgress("installing", 100)
	if err := swapExecutable(exe, newExe); err != nil {
		_ = os.Remove(newExe)
		return err
	}

	relaunch := exec.Command(exe)
	relaunch.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000008} // DETACHED_PROCESS
	if err := relaunch.Start(); err != nil {
		return err
	}

	a.emitUpdateProgress("restarting", 100)
	wailsruntime.Quit(a.ctx)
	return nil
}
