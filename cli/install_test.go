// Copyright 2025 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"testing"
)

func TestExtract(t *testing.T) {
	copy := "zpm:zl:s@te@st"
	result := ExtractInstall(copy)
	if result.Site != "zpm" {
		t.Fatalf("Recieved '%q'. Wanted %q", result.Site, "zpm")
	}

	if result.Package != "zl:s" {
		t.Fatalf("Recieved %q. Wanted %q", result.Package, "zl:s")
	}

	if result.Version != "te@st" {
		t.Fatalf("Recieved %q. Wanted %q", result.Version, "te@st")
	}
}

func TestSitePkg(t *testing.T) {
	copy := "zpm:zl:s"
	result := ExtractInstall(copy)
	if result.Site != "zpm" {
		t.Fatalf("Recieved '%q'. Wanted %q", result.Site, "zpm")
	}

	if result.Package != "zl:s" {
		t.Fatalf("Recieved %q. Wanted %q", result.Package, "zl:s")
	}

	if result.Version != "" {
		t.Fatalf("Recieved %q. Wanted %q", result.Version, "")
	}
}

func TestPkg(t *testing.T) {
	copy := "zls@11"
	result := ExtractInstall(copy)
	if result.Site != "" {
		t.Fatalf("Recieved '%q'. Wanted %q", result.Site, "")
	}

	if result.Package != "zls" {
		t.Fatalf("Recieved %q. Wanted %q", result.Package, "zls")
	}

	if result.Version != "11" {
		t.Fatalf("Recieved %q. Wanted %q", result.Version, "11")
	}
}

