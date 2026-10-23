// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func zon(ver string) string {
	return ".{\n    .name = .demo,\n    .version = \"0.0.1\",\n    .minimum_zig_version = \"" + ver + "\",\n}\n"
}

func buildZigLock(ver string) string {
	return "//! zvm-lock: " + ver + "\nconst std = @import(\"std\");\npub fn build(b: *std.Build) void { _ = b; }\n"
}

// writeTree creates files (relative path -> contents) under root.
func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, contents := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestFindVersionSource(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string
		start    string // relative to the tree root
		wantOK   bool
		wantVer  string
		wantFile string // relative to the tree root
		wantKey  string
	}{
		{
			name:     "zon in start dir",
			files:    map[string]string{"proj/build.zig.zon": zon("0.15.1")},
			start:    "proj",
			wantOK:   true,
			wantVer:  "0.15.1",
			wantFile: "proj/build.zig.zon",
			wantKey:  versionKeyMinimum,
		},
		{
			name:     "zon in parent of start dir",
			files:    map[string]string{"proj/build.zig.zon": zon("0.15.1"), "proj/src/deep/.keep": ""},
			start:    "proj/src/deep",
			wantOK:   true,
			wantVer:  "0.15.1",
			wantFile: "proj/build.zig.zon",
			wantKey:  versionKeyMinimum,
		},
		{
			name: "zvm-lock beats zon in the same dir",
			files: map[string]string{
				"proj/build.zig":     buildZigLock("0.14.1"),
				"proj/build.zig.zon": zon("0.15.1"),
			},
			start:    "proj",
			wantOK:   true,
			wantVer:  "0.14.1",
			wantFile: "proj/build.zig",
			wantKey:  versionKeyLock,
		},
		{
			name: "nearer zon beats farther zvm-lock",
			files: map[string]string{
				"build.zig":           buildZigLock("0.14.1"),
				"sub/build.zig.zon":   zon("0.15.2"),
				"sub/inner/src/.keep": "",
			},
			start:    "sub/inner/src",
			wantOK:   true,
			wantVer:  "0.15.2",
			wantFile: "sub/build.zig.zon",
			wantKey:  versionKeyMinimum,
		},
		{
			name: "build.zig without lock falls through to zon",
			files: map[string]string{
				"proj/build.zig":     "const std = @import(\"std\");\n",
				"proj/build.zig.zon": zon("0.15.1"),
			},
			start:    "proj",
			wantOK:   true,
			wantVer:  "0.15.1",
			wantFile: "proj/build.zig.zon",
			wantKey:  versionKeyMinimum,
		},
		{
			name: "dir without a version keeps searching upward",
			files: map[string]string{
				"build.zig.zon":      zon("0.13.0"),
				"proj/build.zig":     "const std = @import(\"std\");\n",
				"proj/build.zig.zon": ".{ .name = .demo, .version = \"0.0.1\" }\n",
			},
			start:    "proj",
			wantOK:   true,
			wantVer:  "0.13.0",
			wantFile: "build.zig.zon",
			wantKey:  versionKeyMinimum,
		},
		{
			name: "commented minimum_zig_version is ignored",
			files: map[string]string{
				"proj/build.zig.zon": ".{\n    // .minimum_zig_version = \"0.11.0\",\n}\n",
			},
			start:  "proj",
			wantOK: false,
		},
		{
			name: "malformed zvm-lock falls through to zon",
			files: map[string]string{
				"proj/build.zig":     "//! zvm-lock:\n",
				"proj/build.zig.zon": zon("0.15.1"),
			},
			start:    "proj",
			wantOK:   true,
			wantVer:  "0.15.1",
			wantFile: "proj/build.zig.zon",
			wantKey:  versionKeyMinimum,
		},
		{
			name:     "zvm-lock key is case-insensitive and v prefix is trimmed",
			files:    map[string]string{"proj/build.zig": "//! ZVM-Lock: v0.14.0\n"},
			start:    "proj",
			wantOK:   true,
			wantVer:  "0.14.0",
			wantFile: "proj/build.zig",
			wantKey:  versionKeyLock,
		},
		{
			name:     "zvm-lock with leading whitespace",
			files:    map[string]string{"proj/build.zig": "   //!   zvm-lock:   master  \n"},
			start:    "proj",
			wantOK:   true,
			wantVer:  "master",
			wantFile: "proj/build.zig",
			wantKey:  versionKeyLock,
		},
		{
			name:     "minimum_zig_version v prefix is trimmed",
			files:    map[string]string{"proj/build.zig.zon": zon("v0.16.0-dev.1334+06d08daba")},
			start:    "proj",
			wantOK:   true,
			wantVer:  "0.16.0-dev.1334+06d08daba",
			wantFile: "proj/build.zig.zon",
			wantKey:  versionKeyMinimum,
		},
		{
			name:   "no version files at all",
			files:  map[string]string{"proj/src/main.zig": "pub fn main() void {}\n"},
			start:  "proj/src",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			writeTree(t, root, tt.files)

			src, ok, err := FindVersionSource(filepath.Join(root, filepath.FromSlash(tt.start)))
			if err != nil {
				t.Fatalf("FindVersionSource: %v", err)
			}
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v (src %+v)", ok, tt.wantOK, src)
			}
			if !ok {
				return
			}
			if src.Version != tt.wantVer {
				t.Errorf("Version = %q, want %q", src.Version, tt.wantVer)
			}
			if want := filepath.Join(root, filepath.FromSlash(tt.wantFile)); src.File != want {
				t.Errorf("File = %q, want %q", src.File, want)
			}
			if src.Key != tt.wantKey {
				t.Errorf("Key = %q, want %q", src.Key, tt.wantKey)
			}
		})
	}
}

// ExtractMinimumZigVersion backs `zvm use` and `zvm run` with no version
// argument; it must keep reading only the current directory.
func TestExtractMinimumZigVersionIn(t *testing.T) {
	tests := []struct {
		name    string
		files   map[string]string
		dir     string
		want    string
		wantErr bool
	}{
		{name: "in dir", files: map[string]string{"build.zig.zon": zon("0.15.1")}, dir: ".", want: "0.15.1"},
		{name: "parent is not searched", files: map[string]string{"build.zig.zon": zon("0.15.1"), "sub/.keep": ""}, dir: "sub", wantErr: true},
		{name: "unparsable", files: map[string]string{"build.zig.zon": ".{}\n"}, dir: ".", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			writeTree(t, root, tt.files)

			got, err := ExtractMinimumZigVersionIn(filepath.Join(root, tt.dir))
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
