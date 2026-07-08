package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// UpdateProgress is streamed to the frontend over the "update:progress"
// event while ApplyUpdate runs.
type UpdateProgress struct {
	Phase   string `json:"phase"` // "downloading" | "installing" | "restarting"
	Percent int    `json:"percent"`
}

func (a *App) emitUpdateProgress(phase string, percent int) {
	wailsruntime.EventsEmit(a.ctx, "update:progress", UpdateProgress{Phase: phase, Percent: percent})
}

// ApplyUpdate downloads the latest release and installs it in place, then
// restarts TaskMax. The frontend falls back to opening the releases page in
// the browser when this returns an error.
func (a *App) ApplyUpdate() error {
	if version == "dev" {
		return errors.New("dev builds cannot self-update")
	}
	release, err := fetchLatestRelease()
	if err != nil {
		return err
	}
	if !semverLess(version, release.TagName) {
		return errors.New("already up to date")
	}
	return a.applyUpdate(release)
}

// downloadAsset streams a release asset to dest, reporting download progress
// to the frontend. The destination directory must already exist.
func (a *App) downloadAsset(asset *releaseAsset, dest string) error {
	client := &http.Client{Timeout: 15 * time.Minute} // whole-download budget
	req, err := http.NewRequest(http.MethodGet, asset.URL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "TaskMax-update")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: %s", resp.Status)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	total := resp.ContentLength
	if total <= 0 {
		total = asset.Size
	}
	var done int64
	lastPct := -1
	buf := make([]byte, 128*1024)
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := out.Write(buf[:n]); werr != nil {
				return werr
			}
			done += int64(n)
			if total > 0 {
				if pct := int(done * 100 / total); pct != lastPct {
					lastPct = pct
					a.emitUpdateProgress("downloading", pct)
				}
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return rerr
		}
	}
	return out.Sync()
}

// swapExecutable atomically replaces the running executable with newPath
// (which must be on the same volume): the live binary is renamed aside —
// allowed on every OS we ship to — and the new one takes its place.
func swapExecutable(exe, newPath string) error {
	old := exe + ".old"
	_ = os.Remove(old) // leftover from a previous update
	if err := os.Rename(exe, old); err != nil {
		return fmt.Errorf("could not move the current binary aside (is the install directory writable?): %w", err)
	}
	if err := os.Rename(newPath, exe); err != nil {
		// Try to put things back so the app still launches next time.
		_ = os.Rename(old, exe)
		return err
	}
	return nil
}

// cleanupUpdateArtifacts removes the renamed-aside binary a previous
// self-update left behind. Called once on startup; failures are harmless
// (the file stays locked briefly while the old process exits) and retried
// on the next launch.
func cleanupUpdateArtifacts() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	old := exe + ".old"
	for i := 0; i < 5; i++ {
		if err := os.Remove(old); err == nil || os.IsNotExist(err) {
			return
		}
		time.Sleep(time.Second)
	}
}

// updateTempDir returns a scratch directory for downloaded update files.
func updateTempDir() (string, error) {
	dir := filepath.Join(os.TempDir(), "taskmax-update")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}
