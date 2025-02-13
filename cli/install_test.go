// Copyright 2025 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"os"
	"os/exec"
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

// verification for XDG and ZVM_PATH install checks
func verifyInstallation(t *testing.T, zvm *ZVM) {
	// Verify zig binary exists in state directory
	zigPath := filepath.Join(zvm.Directories.state, "0.13.0", "zig")
	if runtime.GOOS == "windows" {
		zigPath += ".exe"
	}
	if _, err := os.Stat(zigPath); os.IsNotExist(err) {
		t.Errorf("Zig binary not found at expected location: %s", zigPath)
	}

	// Verify zls binary exists in state directory
	zlsPath := filepath.Join(zvm.Directories.state, "0.13.0", "zls")
	if runtime.GOOS == "windows" {
		zlsPath += ".exe"
	}
	if _, err := os.Stat(zlsPath); os.IsNotExist(err) {
		t.Errorf("ZLS binary not found at expected location: %s", zlsPath)
	}

	// Verify symlinks in bin directory
	zigLink := filepath.Join(zvm.Directories.bin, "zig")
	if runtime.GOOS == "windows" {
		zigLink += ".exe"
	}
	if _, err := os.Stat(zigLink); os.IsNotExist(err) {
		t.Errorf("Zig symlink not found at expected location: %s", zigLink)
		t.Error("Bin directory contents:")
		listFiles(t, zvm.Directories.bin, "\t")
	}

	zlsLink := filepath.Join(zvm.Directories.bin, "zls")
	if runtime.GOOS == "windows" {
		zlsLink += ".exe"
	}
	if _, err := os.Stat(zlsLink); os.IsNotExist(err) {
		t.Errorf("ZLS symlink not found at expected location: %s", zlsLink)
	}
}

func listFiles(t *testing.T, path string, indent string) error {
	files, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	for _, file := range files {
		fullPath := filepath.Join(path, file.Name())

		// Get file info to check if it's a symlink
		info, err := file.Info()
		if err != nil {
			t.Errorf("error getting file info for %s: %v", fullPath, err)
		}
		switch {
		case file.IsDir():
			t.Errorf("%s📁 %s\n", indent, file.Name())
			err := listFiles(t, fullPath, indent+"  ")
			if err != nil {
				return err
			}
		case info.Mode()&os.ModeSymlink != 0:
			target, err := os.Readlink(fullPath)
			if err != nil {
				t.Errorf("%s🔗 %s (unable to read target)\n", indent, file.Name())
			} else {
				t.Errorf("%s🔗 %s -> %s\n", indent, file.Name(), target)
			}
		default:
			t.Errorf("%s📄 %s\n", indent, file.Name())
		}
	}
	return nil
}

// used for XDG and ZVM_PATH
func performInstallation(t *testing.T, zvm *ZVM) {
	// Install Zig 0.13.0
	if err := zvm.Install("0.13.0", true); err != nil {
		t.Fatal(err)
	}

	// Install ZLS
	if err := zvm.InstallZls("0.13.0", "safe", true); err != nil {
		t.Fatal(err)
	}
}

func TestXDGInstall(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping download test in short mode")
	}

	zvmInstall := os.Getenv("ZVM_INSTALL")
	if zvmInstall != "" {
		os.Unsetenv("ZVM_INSTALL")
		defer func() {
			os.Setenv("ZVM_INSTALL", zvmInstall)
		}()
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
		Directories: Directories{
			data:  filepath.Join(xdgDataHome, "zvm"),
			state: filepath.Join(xdgStateHome, "zvm"),
			bin:   xdgBinHome,
			cache: filepath.Join(xdgCacheHome, "zvm"),
		},
	}

	performInstallation(t, zvm)
	verifyInstallation(t, zvm)
}

func TestZVMPathInstall(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping download test in short mode")
	}

	zvmInstall := os.Getenv("ZVM_INSTALL")
	if zvmInstall != "" {
		os.Unsetenv("ZVM_INSTALL")
		defer func() {
			os.Setenv("ZVM_INSTALL", zvmInstall)
		}()
	}
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "zvm-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Set ZVM_PATH environment variable
	os.Setenv("ZVM_PATH", tmpDir)
	defer os.Unsetenv("ZVM_PATH")

	t.Logf("ZVM_PATH %s", tmpDir)

	// Initialize ZVM
	zvm := &ZVM{
		Directories: Directories{
			data:  filepath.Join(tmpDir, "data"),
			state: filepath.Join(tmpDir, "state"),
			bin:   filepath.Join(tmpDir, "bin"),
			cache: filepath.Join(tmpDir, "cache"),
		},
	}

	performInstallation(t, zvm)
	verifyInstallation(t, zvm)
}
func TestInstallationScript(t *testing.T) {
	_, currentFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Dir(filepath.Dir(currentFile))
	var script struct {
		name     string
		path     string
		executor string
	}

	switch runtime.GOOS {
	case "windows":
		script = struct {
			name     string
			path     string
			executor string
		}{"install.ps1", filepath.Join(repoRoot, "install.ps1"), "powershell"}
	default:
		script = struct {
			name     string
			path     string
			executor string
		}{"install.sh", filepath.Join(repoRoot, "install.sh"), "/bin/bash"}
	}

	zvmInstall := os.Getenv("ZVM_INSTALL")
	if zvmInstall != "" {
		os.Unsetenv("ZVM_INSTALL")
		defer func() {
			os.Setenv("ZVM_INSTALL", zvmInstall)
		}()
	}
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "zvm-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Set HOME environment variable to our temp directory for testing
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

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

	// Compile the code, so we have a binary...
	zvmBinary := filepath.Join(repoRoot, "zvm")
	if runtime.GOOS == "windows" {
		zvmBinary += ".exe"
	}
	os.Remove(zvmBinary)
	cmd := exec.Command("go", "build", "-o", zvmBinary)
	cmd.Dir = repoRoot
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build zvm: %v\nOutput: %s", err, output)
	}
	defer os.Remove(zvmBinary)

	// Set XDG environment variables
	os.Setenv("XDG_DATA_HOME", xdgDataHome)
	os.Setenv("XDG_STATE_HOME", xdgStateHome)
	os.Setenv("XDG_BIN_HOME", xdgBinHome)
	os.Setenv("XDG_CACHE_HOME", xdgCacheHome)
	os.Setenv("ZVM_BINARY_ON_DISK_LOCATION", filepath.Dir(zvmBinary))
	// Append XDG_BIN_HOME to PATH
	originalPath := os.Getenv("PATH")
	newPath := originalPath + string(os.PathListSeparator) + xdgBinHome
	os.Setenv("PATH", newPath)

	defer func() {
		os.Unsetenv("XDG_DATA_HOME")
		os.Unsetenv("XDG_STATE_HOME")
		os.Unsetenv("XDG_BIN_HOME")
		os.Unsetenv("XDG_CACHE_HOME")
		os.Unsetenv("ZVM_BINARY_ON_DISK_LOCATION")
		os.Setenv("PATH", originalPath)
	}()

	t.Logf("HOME  %s", os.Getenv("HOME"))
	t.Logf("XDG_DATA_HOME  %s", xdgDataHome)
	t.Logf("XDG_STATE_HOME %s", xdgStateHome)
	t.Logf("XDG_BIN_HOME   %s", xdgBinHome)
	t.Logf("XDG_CACHE_HOME %s", xdgCacheHome)

	// Run the installation script
	cmd = exec.Command(script.executor, script.path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("installation failed: %v\nOutput: %s", err, output)
	}
	t.Logf("Output: %s", output)

	// Create a ZVM instance to get expected paths
	zvmDirectories := zvmDirectories(tmpDir, true)

	// Verify only the self directory exists and contains the zvm binary
	selfDir := zvmDirectories.self
	if _, err := os.Stat(selfDir); os.IsNotExist(err) {
		t.Errorf("self directory not found: %s", selfDir)
	}

	binaryName := "zvm"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	if _, err := os.Stat(filepath.Join(selfDir, binaryName)); os.IsNotExist(err) {
		t.Errorf("%s binary not found in self directory", binaryName)
	}
}
