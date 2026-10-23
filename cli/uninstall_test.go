package cli

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestUninstallRejectsUnsafeTargets(t *testing.T) {
	for _, target := range []string{"", ".", "..", "settings.json", "self", "bin", "SELF", "notes.txt", "0.15.0/zig", "0.15.0/", "./0.15.0", "../outside", "0.15.0/../../outside", `/tmp/outside`, `0.15.0\zig`, `C:outside`} {
		t.Run(target, func(t *testing.T) {
			z := &ZVM{baseDir: t.TempDir()}
			for _, dir := range []string{"self", "bin", "0.15.0", "SELF"} {
				if err := os.MkdirAll(filepath.Join(z.baseDir, dir), 0700); err != nil {
					t.Fatal(err)
				}
			}
			for _, file := range []string{"settings.json", "notes.txt", "self/keep", "bin/keep", "SELF/keep", "0.15.0/zig"} {
				if err := os.WriteFile(filepath.Join(z.baseDir, file), []byte("keep"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if err := z.Uninstall(target); !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("Uninstall(%q) = %v; want ErrInvalidInput", target, err)
			}
			for _, file := range []string{"settings.json", "notes.txt", "self/keep", "bin/keep", "SELF/keep", "0.15.0/zig"} {
				if _, err := os.Stat(filepath.Join(z.baseDir, file)); err != nil {
					t.Errorf("protected file %s: %v", file, err)
				}
			}
		})
	}
}

func TestUninstallRemovesFullInstallation(t *testing.T) {
	for _, target := range []string{"0.15.1", "0.15", ".15", "stable", "master"} {
		t.Run(target, func(t *testing.T) {
			z := &ZVM{baseDir: t.TempDir()}
			version := "0.15.1"
			if target == "master" {
				version = target
			}
			for _, dir := range []string{version + "/lib", "0.14.0"} {
				if err := os.MkdirAll(filepath.Join(z.baseDir, dir), 0700); err != nil {
					t.Fatal(err)
				}
			}
			writeZig(t, filepath.Join(z.baseDir, version))
			if err := os.WriteFile(filepath.Join(z.baseDir, version, "lib", "keep"), []byte("data"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := z.Uninstall(target); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(filepath.Join(z.baseDir, version)); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("installation still exists: %v", err)
			}
			if _, err := os.Stat(filepath.Join(z.baseDir, "0.14.0")); err != nil {
				t.Fatalf("other installation: %v", err)
			}
		})
	}
}

func TestUninstallRejectsSymlinkTarget(t *testing.T) {
	z := &ZVM{baseDir: t.TempDir()}
	outside := t.TempDir()
	marker := filepath.Join(outside, "keep")
	if err := os.WriteFile(marker, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(z.baseDir, "0.15.0")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := z.Uninstall("0.15.0"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("got %v; want ErrInvalidInput", err)
	}
	if _, err := os.Lstat(link); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatal(err)
	}
}

func TestUninstallRemovesActiveBinLink(t *testing.T) {
	z := &ZVM{baseDir: t.TempDir()}
	for _, ver := range []string{"0.14.0", "0.15.1"} {
		if err := os.MkdirAll(filepath.Join(z.baseDir, ver), 0700); err != nil {
			t.Fatal(err)
		}
		writeZig(t, filepath.Join(z.baseDir, ver))
	}

	if err := z.setBin("0.15.1"); err != nil {
		t.Fatal(err)
	}

	// Uninstalling an inactive version keeps the active bin link intact.
	if err := z.Uninstall("0.14.0"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(z.baseDir, "bin")); err != nil {
		t.Fatalf("bin link should remain after uninstalling inactive version: %v", err)
	}

	// Uninstalling the active version removes the bin link so it is not left dangling.
	if err := z.Uninstall("0.15.1"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(z.baseDir, "bin")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected active bin link to be removed, got err=%v", err)
	}
}

// Uninstall must only remove directories that hold a Zig installation.
func TestUninstallOnlyRemovesZigInstalls(t *testing.T) {
	tests := []struct {
		name    string
		arg     string
		setup   func(t *testing.T, base string)
		target  string // path under base that must (not) survive
		wantErr bool
		removed bool
	}{
		{
			name:    "zig install",
			arg:     "0.15.1",
			setup:   func(t *testing.T, base string) { writeZig(t, filepath.Join(base, "0.15.1")) },
			target:  "0.15.1",
			removed: true,
		},
		{
			name: "windows zig install",
			arg:  "0.15.1",
			setup: func(t *testing.T, base string) {
				if err := os.MkdirAll(filepath.Join(base, "0.15.1"), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(base, "0.15.1", "zig.exe"), []byte("x"), 0600); err != nil {
					t.Fatal(err)
				}
			},
			target:  "0.15.1",
			removed: true,
		},
		{
			name: "folder without zig",
			arg:  "my-notes",
			setup: func(t *testing.T, base string) {
				if err := os.MkdirAll(filepath.Join(base, "my-notes"), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(base, "my-notes", "important.txt"), []byte("x"), 0600); err != nil {
					t.Fatal(err)
				}
			},
			target:  "my-notes",
			wantErr: true,
		},
		{
			name: "zig is a directory",
			arg:  "fake",
			setup: func(t *testing.T, base string) {
				if err := os.MkdirAll(filepath.Join(base, "fake", "zig"), 0700); err != nil {
					t.Fatal(err)
				}
			},
			target:  "fake",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := t.TempDir()
			tt.setup(t, base)
			err := (&ZVM{baseDir: base}).Uninstall(tt.arg)
			if tt.wantErr != errors.Is(err, ErrInvalidInput) {
				t.Fatalf("Uninstall(%q) = %v, wantErr %v", tt.arg, err, tt.wantErr)
			}
			if !tt.wantErr && err != nil {
				t.Fatal(err)
			}
			_, statErr := os.Lstat(filepath.Join(base, tt.target))
			if removed := errors.Is(statErr, os.ErrNotExist); removed != tt.removed {
				t.Fatalf("Uninstall(%q): %s removed = %v, want %v", tt.arg, tt.target, removed, tt.removed)
			}
		})
	}
}

func writeZig(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "zig"), []byte("#!/bin/sh\n"), 0700); err != nil {
		t.Fatal(err)
	}
}
