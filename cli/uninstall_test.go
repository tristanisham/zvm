// Copyright 2025 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUninstallOnlyRemovesZigInstalls(t *testing.T) {
	tests := []struct {
		name        string
		arg         string
		setup       func(t *testing.T, base string)
		target      string // path under base that must (not) survive
		wantRemoved bool
	}{
		{
			name: "zig install",
			arg:  "0.15.1",
			setup: func(t *testing.T, base string) {
				writeFile(t, filepath.Join(base, "0.15.1", "zig"))
				writeFile(t, filepath.Join(base, "0.15.1", "lib", "std", "std.zig"))
			},
			target:      "0.15.1",
			wantRemoved: true,
		},
		{
			name: "windows zig install",
			arg:  "0.15.1",
			setup: func(t *testing.T, base string) {
				writeFile(t, filepath.Join(base, "0.15.1", "zig.exe"))
			},
			target:      "0.15.1",
			wantRemoved: true,
		},
		{
			name: "folder without zig",
			arg:  "my-notes",
			setup: func(t *testing.T, base string) {
				writeFile(t, filepath.Join(base, "my-notes", "important.txt"))
			},
			target: "my-notes",
		},
		{
			name: "self",
			arg:  "self",
			setup: func(t *testing.T, base string) {
				writeFile(t, filepath.Join(base, "self", "zig"))
			},
			target: "self",
		},
		{
			name: "settings file",
			arg:  "settings.json",
			setup: func(t *testing.T, base string) {
				writeFile(t, filepath.Join(base, "settings.json"))
			},
			target: "settings.json",
		},
		{
			name: "subdirectory of an install",
			arg:  "0.15.1/lib",
			setup: func(t *testing.T, base string) {
				writeFile(t, filepath.Join(base, "0.15.1", "zig"))
				writeFile(t, filepath.Join(base, "0.15.1", "lib", "zig"))
			},
			target: filepath.Join("0.15.1", "lib"),
		},
		{
			name: "zig as a directory",
			arg:  "fake",
			setup: func(t *testing.T, base string) {
				if err := os.MkdirAll(filepath.Join(base, "fake", "zig"), 0o755); err != nil {
					t.Fatal(err)
				}
			},
			target: "fake",
		},
		{
			name: "symlinked install",
			arg:  "linked",
			setup: func(t *testing.T, base string) {
				writeFile(t, filepath.Join(base, "0.15.1", "zig"))
				if err := os.Symlink(filepath.Join(base, "0.15.1"), filepath.Join(base, "linked")); err != nil {
					t.Skip("symlinks unavailable:", err)
				}
			},
			target: filepath.Join("0.15.1", "zig"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := t.TempDir()
			tt.setup(t, base)
			z := &ZVM{baseDir: base}

			if err := z.Uninstall(tt.arg); err != nil {
				t.Fatalf("Uninstall(%q) returned error: %v", tt.arg, err)
			}

			_, err := os.Lstat(filepath.Join(base, tt.target))
			removed := os.IsNotExist(err)
			if removed != tt.wantRemoved {
				t.Fatalf("Uninstall(%q): %s removed = %v, want %v", tt.arg, tt.target, removed, tt.wantRemoved)
			}
		})
	}
}

func writeFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
}
