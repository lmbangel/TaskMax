//go:build !windows

package main

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const assetLinux = "TaskMax-linux-amd64.tar.gz"

// macOS ships as an .app bundle inside a zip — replacing a running bundle
// in place needs more care than a binary swap, so it falls back to the
// browser download for now.
func selfUpdateSupported() bool { return runtime.GOOS == "linux" }

// applyUpdate on Linux downloads the release tarball, extracts the TaskMax
// binary over the running one, and relaunches.
func (a *App) applyUpdate(release *releaseInfo) error {
	if runtime.GOOS != "linux" {
		return errors.New("self-update is not supported on this platform yet")
	}
	asset := release.asset(assetLinux)
	if asset == nil {
		return fmt.Errorf("release %s has no Linux asset", release.TagName)
	}

	exe, err := os.Executable()
	if err != nil {
		return err
	}
	dir, err := updateTempDir()
	if err != nil {
		return err
	}
	tarball := filepath.Join(dir, asset.Name)
	if err := a.downloadAsset(asset, tarball); err != nil {
		return err
	}

	a.emitUpdateProgress("installing", 100)

	// Extract the binary next to the running exe so the swap rename stays
	// on one filesystem.
	newExe := exe + ".new"
	if err := extractBinaryFromTarGz(tarball, "TaskMax", newExe); err != nil {
		return err
	}
	if err := swapExecutable(exe, newExe); err != nil {
		_ = os.Remove(newExe)
		return err
	}

	if err := exec.Command(exe).Start(); err != nil {
		return err
	}
	a.emitUpdateProgress("restarting", 100)
	wailsruntime.Quit(a.ctx)
	return nil
}

// extractBinaryFromTarGz pulls a single named file out of a .tar.gz archive
// and writes it to dest with the executable bit set.
func extractBinaryFromTarGz(archive, name, dest string) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if filepath.Base(hdr.Name) != name || hdr.Typeflag != tar.TypeReg {
			continue
		}
		out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, tr); err != nil {
			out.Close()
			return err
		}
		return out.Close()
	}
	return fmt.Errorf("archive does not contain %q", name)
}
