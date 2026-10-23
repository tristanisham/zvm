// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestAlwaysInstallZLSSettings(t *testing.T) {
	if DefaultSettings.AlwaysInstallZLS {
		t.Fatal("AlwaysInstallZLS should be disabled by default")
	}

	settingsPath := filepath.Join(t.TempDir(), "settings.json")
	settings := Settings{path: settingsPath}

	settings.SetAlwaysInstallZLS(true)
	if !settings.AlwaysInstallZLS {
		t.Fatal("SetAlwaysInstallZLS(true) did not enable AlwaysInstallZLS")
	}
	assertPersistedAlwaysInstallZLS(t, settingsPath, true)

	settings.ToggleAlwaysInstallZls()
	if settings.AlwaysInstallZLS {
		t.Fatal("ToggleAlwaysInstallZls did not disable AlwaysInstallZLS")
	}
	assertPersistedAlwaysInstallZLS(t, settingsPath, false)
}

func TestAutoDedupeSetting(t *testing.T) {
	if DefaultSettings.AutoDedupe {
		t.Fatal("AutoDedupe should be disabled by default")
	}

	settingsPath := filepath.Join(t.TempDir(), "settings.json")
	settings := Settings{path: settingsPath}

	for _, want := range []bool{true, false} {
		if err := settings.SetAutoDedupe(want); err != nil {
			t.Fatal(err)
		}
		contents, err := os.ReadFile(settingsPath)
		if err != nil {
			t.Fatal(err)
		}
		var persisted Settings
		if err := json.Unmarshal(contents, &persisted); err != nil {
			t.Fatal(err)
		}
		if persisted.AutoDedupe != want {
			t.Fatalf("persisted AutoDedupe = %v, want %v", persisted.AutoDedupe, want)
		}
	}
}

func TestSettingsSaveUsesReadableFileMode(t *testing.T) {
	settingsPath := filepath.Join(t.TempDir(), "settings.json")
	settings := Settings{path: settingsPath}

	if err := settings.save(); err != nil {
		t.Fatalf("save settings: %v", err)
	}

	info, err := os.Stat(settingsPath)
	if err != nil {
		t.Fatalf("stat settings: %v", err)
	}
	if got := info.Mode().Perm(); runtime.GOOS != "windows" && got != 0o644 {
		t.Fatalf("settings mode = %o, want 644", got)
	}

	// A file left executable by an older release is corrected in place.
	if err := os.Chmod(settingsPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := settings.save(); err != nil {
		t.Fatalf("save settings: %v", err)
	}

	info, err = os.Stat(settingsPath)
	if err != nil {
		t.Fatalf("stat settings: %v", err)
	}
	if got := info.Mode().Perm(); runtime.GOOS != "windows" && got != 0o644 {
		t.Fatalf("settings mode after rewrite = %o, want 644", got)
	}
}

func assertPersistedAlwaysInstallZLS(t *testing.T, path string, want bool) {
	t.Helper()

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read settings: %v", err)
	}

	var persisted Settings
	if err := json.Unmarshal(contents, &persisted); err != nil {
		t.Fatalf("decode settings: %v", err)
	}
	if persisted.AlwaysInstallZLS != want {
		t.Fatalf("persisted AlwaysInstallZLS = %v, want %v", persisted.AlwaysInstallZLS, want)
	}
}
