// Copyright 2025 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestExtract(t *testing.T) {
	copy := "zpm:zl:s@te@st"
	result := ExtractInstall(copy)
	if result.Site != "zpm" {
		t.Fatalf("Recieved '%q'. Wanted %q", result.Site, "zpm")
	}

	if result.Package != "zl:s" {
		t.Fatalf("Recieved %q. Wanted %q", result.Package, "zl:s")
	}

	if result.Version != "te@st" {
		t.Fatalf("Recieved %q. Wanted %q", result.Version, "te@st")
	}
}

func TestSitePkg(t *testing.T) {
	copy := "zpm:zl:s"
	result := ExtractInstall(copy)
	if result.Site != "zpm" {
		t.Fatalf("Recieved '%q'. Wanted %q", result.Site, "zpm")
	}

	if result.Package != "zl:s" {
		t.Fatalf("Recieved %q. Wanted %q", result.Package, "zl:s")
	}

	if result.Version != "" {
		t.Fatalf("Recieved %q. Wanted %q", result.Version, "")
	}
}

func TestPkg(t *testing.T) {
	copy := "zls@11"
	result := ExtractInstall(copy)
	if result.Site != "" {
		t.Fatalf("Recieved '%q'. Wanted %q", result.Site, "")
	}

	if result.Package != "zls" {
		t.Fatalf("Recieved %q. Wanted %q", result.Package, "zls")
	}

	if result.Version != "11" {
		t.Fatalf("Recieved %q. Wanted %q", result.Version, "11")
	}
}

func TestMirrors(t *testing.T) {
	tarURLs := []string{
		"https://ziglang.org/builds/zig-linux-x86_64-0.14.0-dev.1550+4fba7336a.tar.xz",
		"https://ziglang.org/download/0.13.0/zig-linux-x86_64-0.13.0.tar.xz",
	}

	mirrors := []func(string) (string, error){mirrorHryx, mirrorMachEngine}

	for i, mirror := range mirrors {
		for _, tarURL := range tarURLs {
			t.Logf("requestWithMirror url #%d", i)

			newURL, err := mirror(tarURL)
			if err != nil {
				t.Errorf("%q: %q", ErrDownloadFail, err)
			}

			t.Logf("mirror %d; url: %s", i, newURL)

			tarResp, err := attemptDownload(newURL)
			if err != nil {
				continue
			}

			if tarResp.StatusCode != 200 {
				t.Fail()
			}
		}
	}
}

func TestXDGInstall(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping download test in short mode")
	}
	// Create temporary directory structure for XDG paths
	tmpDir, err := os.MkdirTemp("", "zvm-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Set up XDG directory structure
	xdgDataHome := filepath.Join(tmpDir, "data")
	xdgStateHome := filepath.Join(tmpDir, "state") 
	xdgBinHome := filepath.Join(tmpDir, "bin")
	xdgCacheHome := filepath.Join(tmpDir, "cache")

	// Create directories
	for _, dir := range []string{xdgDataHome, xdgStateHome, xdgBinHome, xdgCacheHome} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Set XDG environment variables
	os.Setenv("XDG_DATA_HOME", xdgDataHome)
	os.Setenv("XDG_STATE_HOME", xdgStateHome) 
	os.Setenv("XDG_BIN_HOME", xdgBinHome)
	os.Setenv("XDG_CACHE_HOME", xdgCacheHome)
	defer func() {
		os.Unsetenv("XDG_DATA_HOME")
		os.Unsetenv("XDG_STATE_HOME")
		os.Unsetenv("XDG_BIN_HOME") 
		os.Unsetenv("XDG_CACHE_HOME")
	}()
	t.Logf("XDG_DATA_HOME  %s", xdgDataHome)
	t.Logf("XDG_STATE_HOME %s", xdgStateHome) 
	t.Logf("XDG_BIN_HOME   %s", xdgBinHome)
	t.Logf("XDG_CACHE_HOME %s", xdgCacheHome)

	// Initialize ZVM
	zvm := &ZVM{
		dataDir: filepath.Join(xdgDataHome, "zvm"),
		stateDir: filepath.Join(xdgStateHome, "zvm"),
		binDir: xdgBinHome,
		cacheDir: filepath.Join(xdgCacheHome, "zvm"),
	}

	// Install Zig 0.13.0
	if err := zvm.Install("0.13.0", true); err != nil {
		t.Fatal(err)
	}

	// Install ZLS
	if err := zvm.InstallZls("0.13.0", "safe", true); err != nil {
		t.Fatal(err)
	}

	// Verify zig binary exists in state directory
	zigPath := filepath.Join(zvm.stateDir, "0.13.0", "zig")
	if runtime.GOOS == "windows" {
		zigPath += ".exe"
	}
	if _, err := os.Stat(zigPath); os.IsNotExist(err) {
		t.Errorf("Zig binary not found at expected location: %s", zigPath)
	}

	// Verify zls binary exists in state directory
	zlsPath := filepath.Join(zvm.stateDir, "0.13.0", "zls")
	if runtime.GOOS == "windows" {
		zlsPath += ".exe"
	}
	if _, err := os.Stat(zlsPath); os.IsNotExist(err) {
		t.Errorf("ZLS binary not found at expected location: %s", zlsPath)
	}

	// Verify symlinks in bin directory
	zigLink := filepath.Join(zvm.binDir, "zig")
	if runtime.GOOS == "windows" {
		zigLink += ".exe"
	}
	if _, err := os.Stat(zigLink); os.IsNotExist(err) {
		t.Errorf("Zig symlink not found at expected location: %s", zigLink)
	}

	zlsLink := filepath.Join(zvm.binDir, "zls")
	if runtime.GOOS == "windows" {
		zlsLink += ".exe"
	}
	if _, err := os.Stat(zlsLink); os.IsNotExist(err) {
		t.Errorf("ZLS symlink not found at expected location: %s", zlsLink)
	}
}
