// Copyright 2022 Tristan Isham. All rights reserved.
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

func TestMirrors(t *testing.T) {
	tarURL := "https://ziglang.org/builds/zig-linux-x86_64-0.14.0-dev.1550+4fba7336a.tar.xz"
	mirrors := []func(string) (string, error){mirrorHryx, mirrorMachEngine}

	for i, mirror := range mirrors {
		t.Logf("requestWithMirror url #%d", i)

		newURL, err := mirror(tarURL)
		if err != nil {
			t.Errorf("%q: %q", ErrDownloadFail, err)
		}

		t.Logf("mirror %d; url: %s", i, newURL)

		tarResp, err := attemptDownload(newURL)
		if err != nil {
			continue
		}

		if tarResp.StatusCode != 200 {
			t.Fail()
		}
	}
}
