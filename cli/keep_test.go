package cli

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
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
	for _, name := range []string{"master", "bin", "self", "store"} {
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

func TestKeepMasterReusesExistingVersionDirectory(t *testing.T) {
	z := newTestAliasZVM(t)
	installFakeMaster(t, z, "0.16.0-dev.1+aaaa")
	installTestVersion(t, z, "0.16.0-dev.1+aaaa")

	plan, err := z.planKeepMaster("old", "0.16.0-dev.2+bbbb")
	if err != nil || plan == nil || !plan.exists {
		t.Fatalf("planKeepMaster() = %+v, %v; want a plan for an existing directory", plan, err)
	}
	if err := z.keepMaster(plan); err != nil {
		t.Fatalf("keepMaster: %v", err)
	}

	// master is left for the install to replace.
	if _, err := os.Stat(filepath.Join(z.baseDir, "master", "zig")); err != nil {
		t.Fatalf("master moved although its version was already installed: %v", err)
	}
	if got, _, _ := z.ResolveAlias(t.Context(), "old"); got != "0.16.0-dev.1+aaaa" {
		t.Fatalf("ResolveAlias(old) = %q; want 0.16.0-dev.1+aaaa", got)
	}
}
