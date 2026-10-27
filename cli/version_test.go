package cli

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestStripExcessSlashes(t *testing.T) {
	old := "https://releases.zigtools.org//v1/zls/select-version"
	new := cleanURL(old)

	if new != "https://releases.zigtools.org/v1/zls/select-version" {
		t.Errorf("expected https://releases.zigtools.org/v1/zls/select-version. Got %s", new)
	}
}

func TestVersionMapFilesUseReadableFileMode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"master": {}}`))
	}))
	defer server.Close()

	z := &ZVM{baseDir: t.TempDir(), Settings: DefaultSettings}
	z.Settings.VersionMapUrl = server.URL
	z.Settings.ZlsVMU = server.URL + "/"

	// Simulate a map left executable by an older release.
	stale := filepath.Join(z.baseDir, "versions.json")
	if err := os.WriteFile(stale, []byte(`{}`), 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := z.fetchVersionMap(); err != nil {
		t.Fatalf("fetchVersionMap: %v", err)
	}
	if _, err := z.fetchZlsTaggedVersionMap(); err != nil {
		t.Fatalf("fetchZlsTaggedVersionMap: %v", err)
	}

	for _, name := range []string{"versions.json", "versions-zls.json"} {
		info, err := os.Stat(filepath.Join(z.baseDir, name))
		if err != nil {
			t.Fatalf("stat %s: %v", name, err)
		}
		if got := info.Mode().Perm(); runtime.GOOS != "windows" && got != 0o644 {
			t.Fatalf("%s mode = %o, want 644", name, got)
		}
	}
}

func TestVersionEndpointsRejectHTTPFailures(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"master": {}}`))
	}))
	defer server.Close()
	z := &ZVM{baseDir: t.TempDir(), Settings: DefaultSettings}
	z.Settings.VersionMapUrl = server.URL
	z.Settings.ZlsVMU = server.URL + "/"
	for name, fetch := range map[string]func() error{
		"zig":        func() error { _, err := z.fetchVersionMap(); return err },
		"zls index":  func() error { _, err := z.fetchZlsTaggedVersionMap(); return err },
		"zls select": func() error { _, err := z.fetchZlsVersionByZigVersion("0.15.0", "full"); return err },
	} {
		t.Run(name, func(t *testing.T) {
			if err := fetch(); err == nil {
				t.Fatal("expected HTTP failure")
			}
		})
	}
	for _, name := range []string{"versions.json", "versions-zls.json"} {
		if _, err := os.Stat(filepath.Join(z.baseDir, name)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("unexpected cache file %s: %v", name, err)
		}
	}
}

func TestVersionMapRejectsOversizedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(make([]byte, maxVersionMapSize+1))
	}))
	defer server.Close()
	var result zigVersionMap
	if _, err := fetchVersionJSON(server.URL, &result); !errors.Is(err, ErrInvalidVersionMap) {
		t.Fatalf("got %v; want ErrInvalidVersionMap", err)
	}
}

func TestVersionMapTimeout(t *testing.T) {
	t.Setenv("ZVM_HTTP_TIMEOUT", "1")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
	}))
	defer server.Close()
	var result zigVersionMap
	_, err := fetchVersionJSON(server.URL, &result)
	var netErr interface{ Timeout() bool }
	if !errors.As(err, &netErr) || !netErr.Timeout() {
		t.Fatalf("got %v; want timeout", err)
	}
}

func TestSelectZlsVersionRejectsMissingChecksum(t *testing.T) {
	arch, osName := zigStyleSysInfo()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `{"0.15.0": {%q: {"tarball": "https://example.com/zls.tar.xz"}}}`, arch+"-"+osName)
	}))
	defer server.Close()
	z := &ZVM{baseDir: t.TempDir(), Settings: DefaultSettings}
	z.Settings.ZlsVMU = server.URL + "/"
	if _, _, _, err := z.SelectZlsVersion("0.15.0", "full"); err == nil {
		t.Fatal("expected missing checksum error")
	}
}

func TestZlsVersionErrorCodeDoesNotPanic(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"code":"bad", "message":"invalid"}`))
	}))
	defer server.Close()
	z := &ZVM{Settings: DefaultSettings}
	z.Settings.ZlsVMU = server.URL + "/"
	if _, err := z.fetchZlsVersionByZigVersion("0.15.0", "full"); !errors.Is(err, ErrNoZlsVersion) {
		t.Fatalf("got %v; want ErrNoZlsVersion", err)
	}
}

func TestZlsVersionBadRequestKeepsServerMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"not a valid version!"}`))
	}))
	defer server.Close()
	z := &ZVM{Settings: DefaultSettings}
	z.Settings.ZlsVMU = server.URL + "/"
	_, err := z.fetchZlsVersionByZigVersion("bogus", "full")
	if !errors.Is(err, ErrNoZlsVersion) || !strings.Contains(err.Error(), "not a valid version!") {
		t.Fatalf("got %v; want ErrNoZlsVersion with server message", err)
	}
}

func TestDownloadTimeout(t *testing.T) {
	tests := []struct {
		env  string
		want time.Duration
	}{
		{"", httpDefaultTimeout},
		{"0", 0},
		{"30", 30 * time.Second},
		{"-5", httpDefaultTimeout},
		{"abc", httpDefaultTimeout},
		{"99999999999999", httpDefaultTimeout},
	}
	for _, tt := range tests {
		t.Setenv("ZVM_HTTP_TIMEOUT", tt.env)
		if got := downloadTimeout(); got != tt.want {
			t.Errorf("ZVM_HTTP_TIMEOUT=%q: got %v; want %v", tt.env, got, tt.want)
		}
	}
}
