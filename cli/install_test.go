// Copyright 2025 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
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

func TestDownloadTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test"))
	}))

	defer server.Close()

	var buf bytes.Buffer
	_, err := attemptDownload(&buf, server.URL, "version_string", false, 500*time.Millisecond)

	if urlError, ok := errors.AsType[*url.Error](err); ok {
		if !urlError.Timeout() {
			t.Fatal("expected the *url.Error to return true for Timeout()")
		}
	} else {
		t.Fatal("expected to be able to interpret the error as a *url.Error")
	}
}
