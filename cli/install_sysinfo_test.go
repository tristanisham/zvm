// Copyright 2025 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"runtime"
	"testing"
)

func TestZigStyleSysInfo(t *testing.T) {
	// Determine expected defaults from this machine's runtime
	defaultArch := runtime.GOARCH
	switch defaultArch {
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
