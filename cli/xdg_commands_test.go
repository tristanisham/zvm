// Copyright 2025 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCleanUsesXDGCacheDirectory(t *testing.T) {
	root := t.TempDir()
	stateDir := filepath.Join(root, "state")
	cacheDir := filepath.Join(root, "cache")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatal(err)
	}

	cacheArchive := filepath.Join(cacheDir, "download.tar")
	stateArchive := filepath.Join(stateDir, "installed.tar")
	for _, path := range []string{cacheArchive, stateArchive} {
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	zvm := ZVM{Directories: Directories{state: stateDir, cache: cacheDir, useXDGSpec: true}}
	if err := zvm.Clean(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(cacheArchive); !os.IsNotExist(err) {
		t.Fatalf("cache archive was not removed: %v", err)
	}
	if _, err := os.Stat(stateArchive); err != nil {
		t.Fatalf("state file was unexpectedly removed: %v", err)
	}
}

func TestInstalledVersionsUseXDGStateDirectory(t *testing.T) {
	root := t.TempDir()
	stateDir := filepath.Join(root, "state")
	defaultDir := filepath.Join(root, "default")
	if err := os.MkdirAll(filepath.Join(stateDir, "0.15.1"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(defaultDir, "0.14.1"), 0755); err != nil {
		t.Fatal(err)
	}

	zvm := ZVM{
		baseDir: defaultDir,
		Directories: Directories{
			state:      stateDir,
			useXDGSpec: true,
		},
	}
	versions, err := zvm.GetInstalledVersions()
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 1 || versions[0] != "0.15.1" {
		t.Fatalf("installed versions = %v, want [0.15.1]", versions)
	}
}
