// Copyright 2025 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import "testing"

// allZigVersions mirrors the real Zig release index as of 2026-04.
var allZigVersions = []string{
	"0.1.1", "0.2.0", "0.3.0", "0.4.0", "0.5.0",
	"0.6.0", "0.7.0", "0.7.1", "0.8.0", "0.8.1",
	"0.9.0", "0.9.1", "0.10.0", "0.10.1",
	"0.11.0", "0.12.0", "0.12.1", "0.13.0",
	"0.14.0", "0.14.1",
	"0.15.0", "0.15.1", "0.15.2",
	"0.16.0",
	"master",
}

func TestResolveVersionShorthand(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		// Exact matches
		{name: "exact 0.12.0", input: "0.12.0", want: "0.12.0"},
		{name: "exact 0.14.1", input: "0.14.1", want: "0.14.1"},
		{name: "exact 0.16.0", input: "0.16.0", want: "0.16.0"},
		{name: "exact 0.1.1", input: "0.1.1", want: "0.1.1"},

		// Minor shorthand — single patch
		{name: "shorthand 0.13", input: "0.13", want: "0.13.0"},
		{name: "shorthand 0.16", input: "0.16", want: "0.16.0"},
		{name: "shorthand 0.11", input: "0.11", want: "0.11.0"},
		{name: "shorthand 0.6", input: "0.6", want: "0.6.0"},

		// Minor shorthand — picks latest patch
		{name: "shorthand 0.15 latest", input: "0.15", want: "0.15.2"},
		{name: "shorthand 0.12 latest", input: "0.12", want: "0.12.1"},
		{name: "shorthand 0.14 latest", input: "0.14", want: "0.14.1"},
		{name: "shorthand 0.7 latest", input: "0.7", want: "0.7.1"},
		{name: "shorthand 0.8 latest", input: "0.8", want: "0.8.1"},
		{name: "shorthand 0.9 latest", input: "0.9", want: "0.9.1"},
		{name: "shorthand 0.10 latest", input: "0.10", want: "0.10.1"},

		// Dot prefix shorthand
		{name: "dot .12", input: ".12", want: "0.12.1"},
		{name: "dot .15", input: ".15", want: "0.15.2"},
		{name: "dot .13", input: ".13", want: "0.13.0"},
		{name: "dot .16", input: ".16", want: "0.16.0"},
		{name: "dot .7", input: ".7", want: "0.7.1"},
		{name: "dot .1", input: ".1", want: "0.1.1"},

		// Special versions
		{name: "master passthrough", input: "master", want: "master"},

		// Stable alias
		{name: "stable resolves to latest release", input: "stable", want: "0.16.0"},

		// No match — returns input unchanged
		{name: "no match 0.99", input: "0.99", want: "0.99"},
		{name: "no match 1.0", input: "1.0", want: "1.0"},

		// Empty input
		{name: "empty string errors", input: "", wantErr: true},

		// Prevents false prefix matches
		{name: "0.1 does not match 0.10.x or 0.11.x etc", input: "0.1", want: "0.1.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveVersionShorthand(tt.input, allZigVersions)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("resolveVersionShorthand(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolveStable(t *testing.T) {
	tests := []struct {
		name      string
		available []string
		want      string
		wantErr   bool
	}{
		{
			name:      "all real versions",
			available: allZigVersions,
			want:      "0.16.0",
		},
		{
			name:      "with dev version",
			available: []string{"0.13.0", "0.14.0-dev.123+abc", "master"},
			want:      "0.13.0",
		},
		{
			name:      "only master",
			available: []string{"master"},
			wantErr:   true,
		},
		{
			name:      "only dev versions",
			available: []string{"master", "0.17.0-dev.76+ff612334f"},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveStable(tt.available)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("resolveStable() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLatestVersion(t *testing.T) {
	tests := []struct {
		name     string
		versions []string
		want     string
	}{
		{name: "single", versions: []string{"0.12.0"}, want: "0.12.0"},
		{name: "multiple", versions: []string{"0.15.0", "0.15.2", "0.15.1"}, want: "0.15.2"},
		{name: "two versions", versions: []string{"0.14.0", "0.14.1"}, want: "0.14.1"},
		{name: "all releases", versions: []string{"0.1.1", "0.16.0", "0.8.0", "0.13.0"}, want: "0.16.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := latestVersion(tt.versions)
			if got != tt.want {
				t.Errorf("latestVersion(%v) = %q, want %q", tt.versions, got, tt.want)
			}
		})
	}
}
