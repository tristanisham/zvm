// Copyright 2025 Tristan Isham. All rights reserved.

package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/tristanisham/zvm/cli/meta"
)

func TestUpgrade(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping upgrade test in short mode")
	}
	// This seems to fail in github actions but works on real hardware?
	// if runtime.GOOS == "darwin" {
	// 	t.Skip("skipping upgrade test temporarily on macos")
	// }

	// Create temporary directory structure for XDG paths
	tmpDir, err := os.MkdirTemp("", "zvm-upgrade-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Set up XDG directory structure similar to install test
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

	zvmInstall := os.Getenv("ZVM_INSTALL")
	if zvmInstall != "" {
		os.Unsetenv("ZVM_INSTALL")
		defer func() {
			os.Setenv("ZVM_INSTALL", zvmInstall)
		}()
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

	// Initialize ZVM with test directories
	zvm := &ZVM{
		Directories: Directories{
			data:  filepath.Join(xdgDataHome, "zvm"),
			state: filepath.Join(xdgStateHome, "zvm"),
			bin:   xdgBinHome,
			cache: filepath.Join(xdgCacheHome, "zvm"),
			self:  xdgBinHome,
		},
	}

	// First verify we can get upgrade information
	tagName, upgradable, err := CanIUpgrade()
	if err != nil {
		t.Fatal(err)
	}
	// Upgradable could be true or false depending on current version
	t.Logf("Current upgrade status - Version: %s, Upgradable: %v", tagName, upgradable)

	// Now we'll set the forceupgrade flag and try again, mostly so we can see that
	// the force is working
	meta.ForceUpgrade = true // ForceUpgrade should only be used for testing
	defer func() { meta.ForceUpgrade = false }()
	tagName, upgradable, err = CanIUpgrade()
	if err != nil {
		t.Fatal(err)
	}
	// Upgradable could be true or false depending on current version
	t.Logf("Current upgrade status - Version: %s, Upgradable: %v", tagName, upgradable)
	if !upgradable {
		t.Error("Expected CanIUpgrade to set upgradable to true, but it is not")
	}

	// Verify the response matches expected format
	if tagName == "" {
		t.Error("Expected non-empty tag name from CanIUpgrade")
	}

	t.Logf("Testing upgrade into %s", zvm.Directories.self)
	// Test the actual upgrade process
	if err := zvm.Upgrade(); err != nil {
		t.Fatal(err)
	}

	// Verify the upgraded binary exists and has correct permissions
	zvmPath := filepath.Join(zvm.Directories.self, "zvm")
	if runtime.GOOS == "windows" {
		zvmPath += ".exe"
	}

	if fi, err := os.Stat(zvmPath); err != nil {
		t.Errorf("Failed to find upgraded binary at %s: %v", zvmPath, err)
	} else {
		// Verify permissions
		if runtime.GOOS != "windows" && fi.Mode().Perm() != 0775 {
			t.Errorf("Expected permissions 0775, got %v", fi.Mode().Perm())
		}
	}
}
