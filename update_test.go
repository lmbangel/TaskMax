package main

import "testing"

func TestSemverLess(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"v0.3.1", "v0.3.2", true},
		{"v0.3.2", "v0.3.1", false},
		{"v0.3.2", "v0.3.2", false},
		{"v0.9.9", "v1.0.0", true},
		{"v1.0.0", "v0.9.9", false},
		{"0.3.1", "v0.3.2", true},      // prefix optional
		{"v0.3.1", "v0.3.2-rc1", true}, // pre-release suffix ignored
		{"v0.3", "v0.3.1", true},       // missing parts count as 0
		{"garbage", "v0.0.1", true},    // unparseable counts as 0.0.0
		{"dev", "dev", false},
	}
	for _, tc := range cases {
		if got := semverLess(tc.a, tc.b); got != tc.want {
			t.Errorf("semverLess(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestParseSemver(t *testing.T) {
	if got := parseSemver(" v1.2.3-beta+build "); got != [3]int{1, 2, 3} {
		t.Errorf("parseSemver = %v, want [1 2 3]", got)
	}
	if got := parseSemver("2.10"); got != [3]int{2, 10, 0} {
		t.Errorf("parseSemver = %v, want [2 10 0]", got)
	}
}

func TestReleaseAssetLookup(t *testing.T) {
	rel := &releaseInfo{Assets: []releaseAsset{
		{Name: "TaskMax-windows-amd64-installer.exe", URL: "http://x/installer"},
		{Name: "TaskMax-linux-amd64.tar.gz", URL: "http://x/linux"},
	}}
	if a := rel.asset("TaskMax-linux-amd64.tar.gz"); a == nil || a.URL != "http://x/linux" {
		t.Errorf("asset lookup returned %+v", a)
	}
	if a := rel.asset("nope.zip"); a != nil {
		t.Errorf("missing asset should return nil, got %+v", a)
	}
}
