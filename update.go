package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// version is stamped by the release workflow via
// -ldflags "-X main.version=v0.1.2". Local builds stay "dev" and never
// report an update.
var version = "dev"

const releasesAPI = "https://api.github.com/repos/lmbangel/TaskMax/releases/latest"

// UpdateInfo is returned to the frontend by CheckForUpdate.
type UpdateInfo struct {
	Available      bool   `json:"available"`
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	URL            string `json:"url"`
}

// GetAppVersion returns the version stamped into this binary.
func (a *App) GetAppVersion() string {
	return version
}

// CheckForUpdate asks GitHub for the latest release and reports whether it
// is newer than the running build. Network errors are returned to the
// caller, which treats them as "no update" — the check must never disturb
// normal use.
func (a *App) CheckForUpdate() (UpdateInfo, error) {
	info := UpdateInfo{CurrentVersion: version}
	if version == "dev" {
		return info, nil
	}

	client := &http.Client{Timeout: 8 * time.Second}
	req, err := http.NewRequest(http.MethodGet, releasesAPI, nil)
	if err != nil {
		return info, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "TaskMax-update-check")

	resp, err := client.Do(req)
	if err != nil {
		return info, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return info, fmt.Errorf("github api returned %s", resp.Status)
	}

	var release struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return info, err
	}

	info.LatestVersion = release.TagName
	info.URL = release.HTMLURL
	info.Available = semverLess(version, release.TagName)
	return info, nil
}

// semverLess reports whether tag a is older than tag b ("v1.2.3" style;
// missing or unparseable parts count as 0).
func semverLess(a, b string) bool {
	pa, pb := parseSemver(a), parseSemver(b)
	for i := 0; i < 3; i++ {
		if pa[i] != pb[i] {
			return pa[i] < pb[i]
		}
	}
	return false
}

func parseSemver(s string) [3]int {
	s = strings.TrimPrefix(strings.TrimSpace(s), "v")
	if i := strings.IndexAny(s, "-+"); i >= 0 {
		s = s[:i]
	}
	var out [3]int
	for i, part := range strings.SplitN(s, ".", 3) {
		n, err := strconv.Atoi(part)
		if err != nil {
			break
		}
		out[i] = n
	}
	return out
}
