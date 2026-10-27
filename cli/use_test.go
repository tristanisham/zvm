package cli

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

func TestSymlinkExists(t *testing.T) {
	link := filepath.Join(t.TempDir(), "symlink.test")
	if err := os.Symlink("use_test.go", link); err != nil {
		t.Error(err)
	}

	stat, err := os.Lstat(link)
	if err != nil {
		t.Errorf("%q: %s", err, stat.Name())
	}
}

func TestUseRejectsUnsafeTargets(t *testing.T) {
	for _, target := range []string{"", ".", "..", "settings.json", "self", "bin", "SELF", "notes.txt", "0.15.0/zig", "0.15.0/", "./0.15.0", "../outside", "0.15.0/../../outside", `/tmp/outside`, `0.15.0\zig`, `C:outside`} {
		t.Run(target, func(t *testing.T) {
			z := &ZVM{baseDir: t.TempDir()}
			for _, dir := range []string{"self", "0.15.0", "SELF"} {
				if err := os.MkdirAll(filepath.Join(z.baseDir, dir), 0700); err != nil {
					t.Fatal(err)
				}
			}
			zigExe := "zig"
			if runtime.GOOS == "windows" {
				zigExe = "zig.exe"
			}
			for _, file := range []string{"settings.json", "notes.txt", "self/keep", "SELF/keep", filepath.Join("0.15.0", zigExe)} {
				if err := os.WriteFile(filepath.Join(z.baseDir, file), []byte("keep"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := z.Use(target); !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("Use(%q) = %v; want ErrInvalidInput", target, err)
			}
		})
	}
}

func TestUseRejectsSymlinkTarget(t *testing.T) {
	z := &ZVM{baseDir: t.TempDir()}
	outside := t.TempDir()
	link := filepath.Join(z.baseDir, "0.15.0")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := z.Use("0.15.0"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("got %v; want ErrInvalidInput", err)
	}
}

func TestUseSwitchesInstalledVersion(t *testing.T) {
	z := &ZVM{baseDir: t.TempDir()}
	zigExe := "zig"
	if runtime.GOOS == "windows" {
		zigExe = "zig.exe"
	}
	installDir := filepath.Join(z.baseDir, "0.15.1")
	if err := os.MkdirAll(installDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(installDir, zigExe), []byte("binary"), 0700); err != nil {
		t.Fatal(err)
	}

	resolved, err := z.Use("0.15")
	if err != nil {
		t.Fatalf("Use(0.15) unexpected error: %v", err)
	}
	if resolved != "0.15.1" {
		t.Fatalf("resolved = %q, want 0.15.1", resolved)
	}

	if _, err := os.Stat(filepath.Join(z.baseDir, "bin", zigExe)); err != nil {
		t.Fatalf("expected active zig binary in bin: %v", err)
	}
}

// A symlink in the base dir to a directory holding zig (e.g. a local build)
// is a usable version: use, ls and run all find it.
func TestUseAcceptsSymlinkToZigBuild(t *testing.T) {
	z := &ZVM{baseDir: t.TempDir()}
	build := t.TempDir()
	if err := os.WriteFile(filepath.Join(build, "zig"), []byte("binary"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(build, filepath.Join(z.baseDir, "local")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	if _, err := z.Use("local"); err != nil {
		t.Fatalf("Use(local) = %v", err)
	}
	if _, err := os.Stat(filepath.Join(z.baseDir, "bin", "zig")); err != nil {
		t.Fatalf("bin does not reach the local build: %v", err)
	}
	if versions, err := z.GetInstalledVersions(); err != nil || !slices.Contains(versions, "local") {
		t.Fatalf("GetInstalledVersions() = %v, %v; want local listed", versions, err)
	}
	if exe, err := z.zigExecutable("local"); err != nil || exe != filepath.Join(z.baseDir, "local", "zig") {
		t.Fatalf("zigExecutable(local) = %q, %v", exe, err)
	}
}

func TestCreateSymlinkPreservesNonLinkBin(t *testing.T) {
	z := &ZVM{baseDir: t.TempDir()}
	bin := filepath.Join(z.baseDir, "bin")
	if err := os.Mkdir(bin, 0700); err != nil {
		t.Fatal(err)
	}
	keep := filepath.Join(bin, "keep")
	if err := os.WriteFile(keep, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	err := z.createSymlink("0.15.0")
	if !errors.Is(err, os.ErrExist) || !strings.Contains(err.Error(), "move it aside") {
		t.Fatalf("got %v; want ErrExist with move-aside hint", err)
	}
	if data, err := os.ReadFile(keep); err != nil || string(data) != "keep" {
		t.Fatalf("bin contents changed: %q, %v", data, err)
	}
}
