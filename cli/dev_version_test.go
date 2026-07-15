// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import "testing"

func TestIsDevelopmentVersion(t *testing.T) {
	tests := []struct {
		version string
		want    bool
	}{
		{"0.16.0-dev.1334+06d08daba", true},
		{"1.2.3-dev.0+abcdef", true},
		{"0.16.0", false},
		{"master", false},
		{"v0.16.0-dev.1334+06d08daba", false},
		{"0.16.0-dev.1334+06D08DABA", false},
		{"0.16.0-dev+06d08daba", false},
		{"0.16.0-dev.1334", false},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			if got := IsDevelopmentVersion(tt.version); got != tt.want {
				t.Errorf("IsDevelopmentVersion(%q) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}

func TestDevelopmentVersionDownloadURL(t *testing.T) {
	const version = "0.16.0-dev.1334+06d08daba"
	tests := []struct {
		name string
		os   string
		arch string
		want string
	}{
		{
			name: "linux tarball",
			os:   "linux",
			arch: "x86_64",
			want: "https://ziglang.org/builds/zig-x86_64-linux-0.16.0-dev.1334+06d08daba.tar.xz",
		},
		{
			name: "macos tarball",
			os:   "macos",
			arch: "aarch64",
			want: "https://ziglang.org/builds/zig-aarch64-macos-0.16.0-dev.1334+06d08daba.tar.xz",
		},
		{
			name: "windows zip",
			os:   "windows",
			arch: "x86_64",
			want: "https://ziglang.org/builds/zig-x86_64-windows-0.16.0-dev.1334+06d08daba.zip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("ZVM_TARGET_OS", tt.os)
			t.Setenv("ZVM_TARGET_ARCH", tt.arch)

			if got := developmentVersionDownloadURL(version); got != tt.want {
				t.Errorf("developmentVersionDownloadURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRequiresUnverifiedDownloadConfirmation(t *testing.T) {
	tests := []struct {
		name       string
		shasum     string
		skipShasum bool
		want       bool
	}{
		{"missing shasum", "", false, true},
		{"missing shasum explicitly skipped", "", true, false},
		{"present shasum", "deadbeef", false, false},
		{"present shasum explicitly skipped", "deadbeef", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := requiresUnverifiedDownloadConfirmation(tt.shasum, tt.skipShasum); got != tt.want {
				t.Errorf("requiresUnverifiedDownloadConfirmation(%q, %v) = %v, want %v", tt.shasum, tt.skipShasum, got, tt.want)
			}
		})
	}
}
