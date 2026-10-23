package cli

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
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
