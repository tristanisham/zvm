// Copyright 2025 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"runtime"
	"testing"
)

func TestZigSystemKeys(t *testing.T) {
	tests := []struct {
		arch string
		os   string
		want []string
	}{
		{arch: "x86", os: "linux", want: []string{"x86-linux", "i386-linux"}},
		{arch: "x86_64", os: "linux", want: []string{"x86_64-linux"}},
		{arch: "aarch64", os: "macos", want: []string{"aarch64-macos"}},
	}

	for _, tt := range tests {
		got := zigSystemKeys(tt.arch, tt.os)
		if len(got) != len(tt.want) {
			t.Fatalf("zigSystemKeys(%q, %q) = %v; want %v", tt.arch, tt.os, got, tt.want)
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Fatalf("zigSystemKeys(%q, %q) = %v; want %v", tt.arch, tt.os, got, tt.want)
			}
		}
	}
}

func TestGetTarPathAcceptsLegacyI386Key(t *testing.T) {
	t.Setenv("ZVM_TARGET_ARCH", "386")
	t.Setenv("ZVM_TARGET_OS", "linux")

	const tarURL = "https://example.com/zig-i386-linux-0.10.0.tar.xz"
	data := map[string]map[string]any{
		"0.10.0": {
			"i386-linux": map[string]any{"tarball": tarURL, "shasum": "abc123"},
		},
	}

	got, err := getTarPath("0.10.0", &data)
	if err != nil {
		t.Fatalf("getTarPath: %v", err)
	}
	if got != tarURL {
		t.Fatalf("tarball = %q; want %q", got, tarURL)
	}

	shasum, err := getVersionShasum("0.10.0", &data)
	if err != nil {
		t.Fatalf("getVersionShasum: %v", err)
	}
	if shasum != "abc123" {
		t.Fatalf("shasum = %q; want abc123", shasum)
	}
}

func TestGetTarPathPrefersModernX86Key(t *testing.T) {
	t.Setenv("ZVM_TARGET_ARCH", "386")
	t.Setenv("ZVM_TARGET_OS", "linux")

	const modernURL = "https://example.com/zig-x86-linux-0.11.0.tar.xz"
	data := map[string]map[string]any{
		"0.11.0": {
			"i386-linux": map[string]any{"tarball": "https://example.com/zig-i386-linux-0.11.0.tar.xz", "shasum": "legacy"},
			"x86-linux":  map[string]any{"tarball": modernURL, "shasum": "modern"},
		},
	}

	got, err := getTarPath("0.11.0", &data)
	if err != nil {
		t.Fatalf("getTarPath: %v", err)
	}
	if got != modernURL {
		t.Fatalf("tarball = %q; want %q", got, modernURL)
	}

	shasum, err := getVersionShasum("0.11.0", &data)
	if err != nil {
		t.Fatalf("getVersionShasum: %v", err)
	}
	if shasum != "modern" {
		t.Fatalf("shasum = %q; want modern", shasum)
	}
}

func TestZigStyleSysInfo(t *testing.T) {
	// Determine expected defaults from this machine's runtime
	defaultArch := runtime.GOARCH
	switch defaultArch {
	case "386":
		defaultArch = "x86"
	case "amd64":
		defaultArch = "x86_64"
	case "arm64":
		defaultArch = "aarch64"
	case "loong64":
		defaultArch = "loongarch64"
	case "ppc64le":
		defaultArch = "powerpc64le"
	}

	defaultOS := runtime.GOOS
	switch defaultOS {
	case "darwin":
		defaultOS = "macos"
	}

	tests := []struct {
		name     string
		envOS    string
		envArch  string
		wantOS   string
		wantArch string
	}{
		{
			name:     "defaults to runtime values",
			envOS:    "",
			envArch:  "",
			wantOS:   defaultOS,
			wantArch: defaultArch,
		},
		{
			name:     "override OS with zig-style name",
			envOS:    "linux",
			envArch:  "",
			wantOS:   "linux",
			wantArch: defaultArch,
		},
		{
			name:     "override arch with zig-style name",
			envOS:    "",
			envArch:  "x86_64",
			wantOS:   defaultOS,
			wantArch: "x86_64",
		},
		{
			name:     "override both with zig-style names",
			envOS:    "freebsd",
			envArch:  "aarch64",
			wantOS:   "freebsd",
			wantArch: "aarch64",
		},
		{
			name:     "override with go-style names maps correctly",
			envOS:    "darwin",
			envArch:  "amd64",
			wantOS:   "macos",
			wantArch: "x86_64",
		},
		{
			name:     "override arch 386 maps to x86",
			envOS:    "",
			envArch:  "386",
			wantOS:   defaultOS,
			wantArch: "x86",
		},
		{
			name:     "override arch arm64 maps to aarch64",
			envOS:    "",
			envArch:  "arm64",
			wantOS:   defaultOS,
			wantArch: "aarch64",
		},
		{
			name:     "override with windows",
			envOS:    "windows",
			envArch:  "amd64",
			wantOS:   "windows",
			wantArch: "x86_64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("ZVM_TARGET_OS", tt.envOS)
			t.Setenv("ZVM_TARGET_ARCH", tt.envArch)

			arch, osName := zigStyleSysInfo()

			if arch != tt.wantArch {
				t.Errorf("arch = %q, want %q", arch, tt.wantArch)
			}
			if osName != tt.wantOS {
				t.Errorf("os = %q, want %q", osName, tt.wantOS)
			}
		})
	}
}
