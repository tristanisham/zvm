// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
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
