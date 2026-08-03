// Copyright 2025 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func mapGetenv(values map[string]string) func(string) string {
	return func(key string) string {
		return values[key]
	}
}

func writeTestSettings(t *testing.T, path string, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestXDGDirectories(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("XDG path resolution is Linux-only")
	}

	home := filepath.Join(string(filepath.Separator), "home", "zvm-test")

	tests := []struct {
		name   string
		env    map[string]string
		config string
		data   string
		state  string
		cache  string
	}{
		{
			name:   "defaults",
			env:    map[string]string{},
			config: filepath.Join(home, ".config", "zvm"),
			data:   filepath.Join(home, ".local", "share", "zvm"),
			state:  filepath.Join(home, ".local", "state", "zvm"),
			cache:  filepath.Join(home, ".cache", "zvm"),
		},
		{
			name: "absolute overrides",
			env: map[string]string{
				"XDG_CONFIG_HOME": "/xdg/config",
				"XDG_DATA_HOME":   "/xdg/data",
				"XDG_STATE_HOME":  "/xdg/state",
				"XDG_CACHE_HOME":  "/xdg/cache",
			},
			config: "/xdg/config/zvm",
			data:   "/xdg/data/zvm",
			state:  "/xdg/state/zvm",
			cache:  "/xdg/cache/zvm",
		},
		{
			name: "relative overrides are ignored",
			env: map[string]string{
				"XDG_CONFIG_HOME": "relative/config",
				"XDG_DATA_HOME":   "relative/data",
				"XDG_STATE_HOME":  "relative/state",
				"XDG_CACHE_HOME":  "relative/cache",
			},
			config: filepath.Join(home, ".config", "zvm"),
			data:   filepath.Join(home, ".local", "share", "zvm"),
			state:  filepath.Join(home, ".local", "state", "zvm"),
			cache:  filepath.Join(home, ".cache", "zvm"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dirs := xdgDirectories(home, mapGetenv(tt.env))
			if dirs.config != tt.config {
				t.Fatalf("config directory = %q, want %q", dirs.config, tt.config)
			}
			if dirs.data != tt.data {
				t.Fatalf("data directory = %q, want %q", dirs.data, tt.data)
			}
			if dirs.state != tt.state {
				t.Fatalf("state directory = %q, want %q", dirs.state, tt.state)
			}
			if dirs.cache != tt.cache {
				t.Fatalf("cache directory = %q, want %q", dirs.cache, tt.cache)
			}
			if dirs.bin != filepath.Join(home, ".local", "bin") {
				t.Fatalf("bin directory = %q", dirs.bin)
			}
		})
	}
}

func TestDiscoverDirectories(t *testing.T) {
	home := t.TempDir()
	xdgConfig := filepath.Join(home, ".config", "zvm", "settings.json")
	writeTestSettings(t, xdgConfig, `{"useXDGSpec":true}`)

	tests := []struct {
		name    string
		goos    string
		env     map[string]string
		useXDG  bool
		rootDir string
	}{
		{
			name:   "linux xdg opt in",
			goos:   "linux",
			env:    map[string]string{},
			useXDG: true,
		},
		{
			name:    "other operating system uses default",
			goos:    "darwin",
			env:     map[string]string{},
			rootDir: filepath.Join(home, ".zvm"),
		},
		{
			name: "zvm path takes precedence",
			goos: "linux",
			env: map[string]string{
				"ZVM_PATH": filepath.Join(home, "custom-zvm"),
			},
			rootDir: filepath.Join(home, "custom-zvm"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dirs, settings, err := discoverDirectories(home, tt.goos, mapGetenv(tt.env))
			if err != nil {
				t.Fatal(err)
			}
			if dirs.useXDGSpec != tt.useXDG {
				t.Fatalf("useXDGSpec = %t, want %t", dirs.useXDGSpec, tt.useXDG)
			}
			if tt.useXDG {
				if settings == nil || !settings.UseXDGSpec {
					t.Fatal("XDG settings were not loaded")
				}
				return
			}
			if dirs.state != tt.rootDir {
				t.Fatalf("default root = %q, want %q", dirs.state, tt.rootDir)
			}
		})
	}
}

func TestDiscoverDirectoriesFallsBackForDisabledOrInvalidXDGSettings(t *testing.T) {
	home := t.TempDir()
	settingsPath := filepath.Join(home, ".config", "zvm", "settings.json")

	writeTestSettings(t, settingsPath, `{"useXDGSpec":false}`)
	dirs, _, err := discoverDirectories(home, "linux", mapGetenv(map[string]string{}))
	if err != nil {
		t.Fatal(err)
	}
	if dirs.useXDGSpec {
		t.Fatal("disabled XDG settings selected the XDG layout")
	}

	writeTestSettings(t, settingsPath, `{`)
	dirs, _, err = discoverDirectories(home, "linux", mapGetenv(map[string]string{}))
	if err == nil {
		t.Fatal("malformed XDG settings did not return an error")
	}
	if dirs.useXDGSpec {
		t.Fatal("malformed XDG settings selected the XDG layout")
	}
}

func TestInitializeUsesXDGWithoutCreatingDefaultDirectory(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("XDG initialization is Linux-only")
	}

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("ZVM_PATH", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")
	writeTestSettings(t, filepath.Join(home, ".config", "zvm", "settings.json"), `{"useXDGSpec":true}`)

	zvm := Initialize()
	if !zvm.usesXDGSpec() {
		t.Fatal("Initialize did not select the XDG layout")
	}
	if !zvm.Settings.UseColor {
		t.Fatal("missing settings fields were not populated from defaults")
	}
	wantSettingsPath := filepath.Join(home, ".config", "zvm", "settings.json")
	if zvm.Settings.path != wantSettingsPath {
		t.Fatalf("settings path = %q, want %q", zvm.Settings.path, wantSettingsPath)
	}
	if _, err := os.Stat(filepath.Join(home, ".zvm")); !os.IsNotExist(err) {
		t.Fatalf("default directory was created during XDG initialization: %v", err)
	}
}

func TestDefaultInstallationWarningUsesDefaultTerminology(t *testing.T) {
	warning := defaultInstallationWarning("/home/test/.zvm")
	if strings.Contains(strings.ToLower(warning), "legacy") {
		t.Fatalf("warning uses legacy terminology: %q", warning)
	}
	if !strings.Contains(warning, "existing default ZVM installation") {
		t.Fatalf("warning does not identify the default installation: %q", warning)
	}
	if !strings.Contains(warning, "rerun install.sh without --use-xdg-spec") {
		t.Fatalf("warning does not explain how to switch back: %q", warning)
	}
}

func TestWarnIfDefaultInstallationExists(t *testing.T) {
	defaultPath := filepath.Join(t.TempDir(), ".zvm")
	if err := os.Mkdir(defaultPath, 0755); err != nil {
		t.Fatal(err)
	}

	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	originalStderr := os.Stderr
	os.Stderr = write
	t.Cleanup(func() {
		os.Stderr = originalStderr
	})

	warnIfDefaultInstallationExists(defaultPath)
	if err := write.Close(); err != nil {
		t.Fatal(err)
	}
	os.Stderr = originalStderr

	output, err := io.ReadAll(read)
	if err != nil {
		t.Fatal(err)
	}
	if string(output) != defaultInstallationWarning(defaultPath) {
		t.Fatalf("warning = %q, want %q", output, defaultInstallationWarning(defaultPath))
	}
}
