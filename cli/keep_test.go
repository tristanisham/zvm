package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// installFakeMaster installs a master build whose zig reports version.
func installFakeMaster(t *testing.T, z *ZVM, version string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake zig is a shell script")
	}
	dir := filepath.Join(z.baseDir, "master")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\necho " + version + "\n"
	if err := os.WriteFile(filepath.Join(dir, "zig"), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
}

// answerAliasPrompt makes confirmAliasUpdate answer yes or no and records
// whether it was asked.
func answerAliasPrompt(t *testing.T, yes bool) *bool {
	t.Helper()
	asked := new(bool)
	orig := confirmAliasUpdate
	confirmAliasUpdate = func(string, string, string) bool { *asked = true; return yes }
	t.Cleanup(func() { confirmAliasUpdate = orig })
	return asked
}

func TestPlanKeepMasterNothingToKeep(t *testing.T) {
	tests := []struct {
		name      string
		installed string // "" for no master install
	}{
		{"master not installed", ""},
		{"master already remote build", "0.16.0-dev.2+bbbb"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			z := newTestAliasZVM(t)
			if tt.installed != "" {
				installFakeMaster(t, z, tt.installed)
			}
			plan, err := z.planKeepMaster("old", "0.16.0-dev.2+bbbb")
			if err != nil || plan != nil {
				t.Fatalf("planKeepMaster() = %+v, %v; want nil, nil", plan, err)
			}
		})
	}
}

func TestPlanKeepMasterRejectsNames(t *testing.T) {
	for _, name := range []string{"master", "MASTER", "bin", "self", "store", "../outside", "bad/name"} {
		t.Run(name, func(t *testing.T) {
			z := newTestAliasZVM(t)
			if _, err := z.planKeepMaster(name, "0.16.0-dev.2+bbbb"); !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("planKeepMaster(%q) error = %v; want ErrInvalidInput", name, err)
			}
		})
	}
}

func TestKeepMasterMovesOldBuildAndSavesAlias(t *testing.T) {
	z := newTestAliasZVM(t)
	installFakeMaster(t, z, "0.16.0-dev.1+aaaa")

	plan, err := z.planKeepMaster("old", "0.16.0-dev.2+bbbb")
	if err != nil || plan == nil {
		t.Fatalf("planKeepMaster() = %+v, %v; want a plan", plan, err)
	}
	if err := z.keepMaster(plan); err != nil {
		t.Fatalf("keepMaster: %v", err)
	}

	if _, err := os.Stat(filepath.Join(z.baseDir, "0.16.0-dev.1+aaaa", "zig")); err != nil {
		t.Fatalf("old build not kept: %v", err)
	}
	if _, err := os.Stat(filepath.Join(z.baseDir, "master")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("master still present after keep: %v", err)
	}
	got, ok, err := z.ResolveAlias(t.Context(), "old")
	if err != nil || !ok || got != "0.16.0-dev.1+aaaa" {
		t.Fatalf("ResolveAlias(old) = %q, %v, %v; want 0.16.0-dev.1+aaaa", got, ok, err)
	}
}

func TestKeepMasterRejectsExistingVersionDirectory(t *testing.T) {
	z := newTestAliasZVM(t)
	installFakeMaster(t, z, "0.16.0-dev.1+aaaa")
	installTestVersion(t, z, "0.16.0-dev.1+aaaa")
	if _, err := z.planKeepMaster("old", "0.16.0-dev.2+bbbb"); !errors.Is(err, os.ErrExist) {
		t.Fatalf("planKeepMaster error = %v; want existing target error", err)
	}
	if _, err := os.Stat(filepath.Join(z.baseDir, "master", "zig")); err != nil {
		t.Fatalf("master was changed: %v", err)
	}
}

func TestKeepMasterRejectsTargetCreatedAfterPlanning(t *testing.T) {
	z := newTestAliasZVM(t)
	installFakeMaster(t, z, "0.16.0-dev.1+aaaa")
	plan, err := z.planKeepMaster("old", "0.16.0-dev.2+bbbb")
	if err != nil {
		t.Fatal(err)
	}
	installTestVersion(t, z, "0.16.0-dev.1+aaaa")
	if err := z.keepMaster(plan); !errors.Is(err, os.ErrExist) {
		t.Fatalf("keepMaster error = %v; want existing target error", err)
	}
}

func TestKeepMasterDeclinedAliasUpdate(t *testing.T) {
	for _, existing := range []string{"other-version", "0.16.0-dev.1+aaaa"} {
		t.Run(existing, func(t *testing.T) {
			asked := answerAliasPrompt(t, false)
			z := newTestAliasZVM(t)
			installFakeMaster(t, z, "0.16.0-dev.1+aaaa")
			if err := z.writeAliases(map[string]string{"old": existing}); err != nil {
				t.Fatal(err)
			}
			if _, err := z.planKeepMaster("old", "0.16.0-dev.2+bbbb"); !errors.Is(err, os.ErrExist) || !*asked {
				t.Fatalf("planKeepMaster error = %v, asked %v; want prompt then existing alias error", err, *asked)
			}
			if err := z.keepMaster(&keptMaster{alias: "old", version: "0.16.0-dev.1+aaaa"}); !errors.Is(err, os.ErrExist) {
				t.Fatalf("keepMaster error = %v; want existing alias error", err)
			}
			if _, err := os.Stat(filepath.Join(z.baseDir, "master", "zig")); err != nil {
				t.Fatalf("master was changed: %v", err)
			}
			got, ok, err := z.ResolveAlias(t.Context(), "old")
			if err != nil || !ok || got != existing {
				t.Fatalf("alias = %q, %v, %v; want %q", got, ok, err, existing)
			}
		})
	}
}

func TestKeepMasterAcceptedAliasUpdate(t *testing.T) {
	answerAliasPrompt(t, true)
	z := newTestAliasZVM(t)
	installFakeMaster(t, z, "0.16.0-dev.1+aaaa")
	if err := z.writeAliases(map[string]string{"old": "other-version"}); err != nil {
		t.Fatal(err)
	}
	plan, err := z.planKeepMaster("old", "0.16.0-dev.2+bbbb")
	if err != nil || plan == nil {
		t.Fatalf("planKeepMaster() = %+v, %v; want a plan", plan, err)
	}
	if err := z.keepMaster(plan); err != nil {
		t.Fatalf("keepMaster: %v", err)
	}
	if got, ok, err := z.ResolveAlias(t.Context(), "old"); err != nil || !ok || got != "0.16.0-dev.1+aaaa" {
		t.Fatalf("alias = %q, %v, %v; want 0.16.0-dev.1+aaaa", got, ok, err)
	}
}

func TestKeepMasterRejectsAliasChangedAfterConfirming(t *testing.T) {
	answerAliasPrompt(t, true)
	z := newTestAliasZVM(t)
	installFakeMaster(t, z, "0.16.0-dev.1+aaaa")
	if err := z.writeAliases(map[string]string{"old": "other-version"}); err != nil {
		t.Fatal(err)
	}
	plan, err := z.planKeepMaster("old", "0.16.0-dev.2+bbbb")
	if err != nil {
		t.Fatal(err)
	}
	if err := z.writeAliases(map[string]string{"old": "third-version"}); err != nil {
		t.Fatal(err)
	}
	if err := z.keepMaster(plan); !errors.Is(err, os.ErrExist) {
		t.Fatalf("keepMaster error = %v; want existing alias error", err)
	}
	if _, err := os.Stat(filepath.Join(z.baseDir, "master", "zig")); err != nil {
		t.Fatalf("master was changed: %v", err)
	}
}

func TestReadConfirmDefaultYes(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"\n", true},
		{"y\n", true},
		{"YES\n", true},
		{"n\n", false},
		{"no\n", false},
		{"maybe\n", false},
		{"", false}, // closed stdin
	}
	for _, tt := range tests {
		if got := readConfirmDefaultYes(strings.NewReader(tt.input)); got != tt.want {
			t.Errorf("readConfirmDefaultYes(%q) = %v; want %v", tt.input, got, tt.want)
		}
	}
}

func TestKeepMasterRejectsAliasCreatedAfterPlanning(t *testing.T) {
	z := newTestAliasZVM(t)
	installFakeMaster(t, z, "0.16.0-dev.1+aaaa")
	plan, err := z.planKeepMaster("old", "0.16.0-dev.2+bbbb")
	if err != nil {
		t.Fatal(err)
	}
	if err := z.writeAliases(map[string]string{"old": "other-version"}); err != nil {
		t.Fatal(err)
	}
	if err := z.keepMaster(plan); !errors.Is(err, os.ErrExist) {
		t.Fatalf("keepMaster error = %v; want existing alias error", err)
	}
	if _, err := os.Stat(filepath.Join(z.baseDir, "master", "zig")); err != nil {
		t.Fatalf("master was changed: %v", err)
	}
}

func TestInstallKeepingMasterIgnoresAliasWhenNothingToKeep(t *testing.T) {
	asked := answerAliasPrompt(t, false)
	const version = "0.16.0-dev.1+aaaa"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprintf(w, `{"master":{"version":%q}}`, version)
	}))
	defer server.Close()
	z := newTestAliasZVM(t)
	z.Settings.VersionMapUrl = server.URL
	installFakeMaster(t, z, version)
	if err := z.writeAliases(map[string]string{"old": "another"}); err != nil {
		t.Fatal(err)
	}
	if _, err := z.InstallKeepingMaster("master", "old", InstallOptions{}); err != nil || *asked {
		t.Fatalf("InstallKeepingMaster error = %v, asked %v; want no error and no prompt", err, *asked)
	}
	if got, ok, err := z.ResolveAlias(t.Context(), "old"); err != nil || !ok || got != "another" {
		t.Fatalf("alias = %q, %v, %v; want unchanged", got, ok, err)
	}
}

func TestInstallKeepingMasterEndToEnd(t *testing.T) {
	if runtime.GOOS == "windows" || zigArchiveExtension() != ".tar.xz" {
		t.Skip("fixture uses a shell script and tar.xz")
	}
	oldVersion := "0.16.0-dev.1+aaaa"
	newVersion := "0.16.0-dev.2+bbbb"
	arch, osName := zigStyleSysInfo()
	stem := fmt.Sprintf("zig-%s-%s-%s", arch, osName, newVersion)
	archive := zigTarball(t, stem, "new master")
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/index.json":
			_ = json.NewEncoder(w).Encode(map[string]any{"master": map[string]any{
				"version":           newVersion,
				arch + "-" + osName: map[string]any{"tarball": server.URL + "/" + stem + ".tar.xz", "shasum": sha256Hex(archive)},
			}})
		case "/" + stem + ".tar.xz":
			_, _ = w.Write(archive)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	z := newTestAliasZVM(t)
	z.Settings.VersionMapUrl = server.URL + "/index.json"
	installFakeMaster(t, z, oldVersion)
	if err := z.setBin("master"); err != nil {
		t.Fatal(err)
	}
	if _, err := z.InstallKeepingMaster("master", "before-upgrade", InstallOptions{}); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(filepath.Join(z.baseDir, oldVersion, "zig")); err != nil || !strings.Contains(string(data), oldVersion) {
		t.Fatalf("old build = %q, %v", data, err)
	}
	if data, err := os.ReadFile(filepath.Join(z.baseDir, "bin", "zig")); err != nil || string(data) != "new master" {
		t.Fatalf("active build = %q, %v", data, err)
	}
	if got, ok, err := z.ResolveAlias(t.Context(), "before-upgrade"); err != nil || !ok || got != oldVersion {
		t.Fatalf("alias = %q, %v, %v", got, ok, err)
	}
}

func TestKeepMasterRejectsMasterReplacedAfterPlanning(t *testing.T) {
	z := newTestAliasZVM(t)
	installFakeMaster(t, z, "0.16.0-dev.1+aaaa")
	plan, err := z.planKeepMaster("old", "0.16.0-dev.3+cccc")
	if err != nil || plan == nil {
		t.Fatalf("planKeepMaster() = %+v, %v; want a plan", plan, err)
	}
	installFakeMaster(t, z, "0.16.0-dev.2+bbbb")
	if err := z.keepMaster(plan); err == nil {
		t.Fatal("keepMaster moved a master build it did not plan for")
	}
	if _, err := os.Stat(filepath.Join(z.baseDir, "0.16.0-dev.1+aaaa")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("master was moved under the planned name: %v", err)
	}
	if _, ok, _ := z.ResolveAlias(t.Context(), "old"); ok {
		t.Fatal("alias saved for a build that was not kept")
	}
}
