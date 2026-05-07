package cli

import (
	"archive/tar"
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ulikunitz/xz"
)

// tarEntry describes one entry to write into an in-memory tar.xz fixture.
type tarEntry struct {
	name     string
	mode     int64
	typeflag byte
	body     []byte
	linkname string
}

// buildTarXZ writes the given entries to a .tar.xz file at path.
func buildTarXZ(t *testing.T, path string, entries []tarEntry) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create fixture: %v", err)
	}
	defer f.Close()

	xzw, err := xz.NewWriter(f)
	if err != nil {
		t.Fatalf("xz writer: %v", err)
	}
	tw := tar.NewWriter(xzw)

	for _, e := range entries {
		hdr := &tar.Header{
			Name:     e.name,
			Mode:     e.mode,
			Size:     int64(len(e.body)),
			Typeflag: e.typeflag,
			Linkname: e.linkname,
		}
		if e.typeflag != tar.TypeReg {
			hdr.Size = 0
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("write tar header %q: %v", e.name, err)
		}
		if e.typeflag == tar.TypeReg && len(e.body) > 0 {
			if _, err := tw.Write(e.body); err != nil {
				t.Fatalf("write tar body %q: %v", e.name, err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := xzw.Close(); err != nil {
		t.Fatalf("close xz writer: %v", err)
	}
}

// canSymlink probes whether the current process can create symlinks.
// On Windows this needs admin or developer mode; the symlink test will
// skip if the probe fails so CI on a stock runner doesn't false-fail.
func canSymlink(t *testing.T) bool {
	t.Helper()
	dir := t.TempDir()
	target := filepath.Join(dir, "t")
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatalf("probe write: %v", err)
	}
	link := filepath.Join(dir, "l")
	if err := os.Symlink(target, link); err != nil {
		return false
	}
	return true
}

// T1: round-trip — encoder + decoder agree on a synthetic fixture.
func TestUntarXZ_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	tarball := filepath.Join(dir, "fixture.tar.xz")
	out := filepath.Join(dir, "out")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}

	want := []byte("hello zig\n")
	buildTarXZ(t, tarball, []tarEntry{
		{name: "pkg/", mode: 0o755, typeflag: tar.TypeDir},
		{name: "pkg/README", mode: 0o644, typeflag: tar.TypeReg, body: want},
	})

	if err := untarXZ(tarball, out); err != nil {
		t.Fatalf("untarXZ: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(out, "pkg", "README"))
	if err != nil {
		t.Fatalf("read extracted: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("contents mismatch: got %q want %q", got, want)
	}
}

// T3: regular-file mode bits survive extraction (verifies executable bit
// for the eventual `zig` binary).
func TestUntarXZ_PreservesExecBit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows mode bits don't map to Unix permissions")
	}
	dir := t.TempDir()
	tarball := filepath.Join(dir, "fixture.tar.xz")
	out := filepath.Join(dir, "out")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}

	buildTarXZ(t, tarball, []tarEntry{
		{name: "bin/", mode: 0o755, typeflag: tar.TypeDir},
		{name: "bin/zig", mode: 0o755, typeflag: tar.TypeReg, body: []byte("#!/bin/sh\necho zig\n")},
		{name: "doc.txt", mode: 0o644, typeflag: tar.TypeReg, body: []byte("doc")},
	})

	if err := untarXZ(tarball, out); err != nil {
		t.Fatalf("untarXZ: %v", err)
	}

	exe, err := os.Stat(filepath.Join(out, "bin", "zig"))
	if err != nil {
		t.Fatalf("stat zig: %v", err)
	}
	if exe.Mode().Perm()&0o111 == 0 {
		t.Fatalf("expected exec bit, got mode %v", exe.Mode().Perm())
	}

	doc, err := os.Stat(filepath.Join(out, "doc.txt"))
	if err != nil {
		t.Fatalf("stat doc: %v", err)
	}
	if doc.Mode().Perm()&0o111 != 0 {
		t.Fatalf("doc should not be executable, got mode %v", doc.Mode().Perm())
	}
}

// T4: relative symlinks (the kind Zig ships under lib/) extract correctly.
func TestUntarXZ_HandlesSymlinks(t *testing.T) {
	if !canSymlink(t) {
		t.Skip("environment cannot create symlinks (Windows without dev mode?)")
	}
	dir := t.TempDir()
	tarball := filepath.Join(dir, "fixture.tar.xz")
	out := filepath.Join(dir, "out")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}

	buildTarXZ(t, tarball, []tarEntry{
		{name: "pkg/", mode: 0o755, typeflag: tar.TypeDir},
		{name: "pkg/real.txt", mode: 0o644, typeflag: tar.TypeReg, body: []byte("payload")},
		{name: "pkg/sub/", mode: 0o755, typeflag: tar.TypeDir},
		{name: "pkg/sub/link.txt", mode: 0o777, typeflag: tar.TypeSymlink, linkname: "../real.txt"},
	})

	if err := untarXZ(tarball, out); err != nil {
		t.Fatalf("untarXZ: %v", err)
	}

	linkPath := filepath.Join(out, "pkg", "sub", "link.txt")
	target, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("readlink: %v", err)
	}
	if filepath.ToSlash(target) != "../real.txt" {
		t.Fatalf("symlink target = %q, want %q (slash-normalized)", target, "../real.txt")
	}

	// Resolution through the symlink only works on platforms where the
	// kernel follows forward-slash separators in link contents. Windows
	// requires backslashes, but .tar.xz extraction only runs on *nix in
	// production (Windows downloads .zip), so this gap is harmless.
	if runtime.GOOS == "windows" {
		return
	}
	got, err := os.ReadFile(linkPath)
	if err != nil {
		t.Fatalf("read via symlink: %v", err)
	}
	if string(got) != "payload" {
		t.Fatalf("read via symlink = %q, want %q", got, "payload")
	}
}

// T5a: zip-slip via a malicious tarball entry name is rejected.
func TestUntarXZ_RejectsZipSlipEntry(t *testing.T) {
	dir := t.TempDir()
	tarball := filepath.Join(dir, "evil.tar.xz")
	out := filepath.Join(dir, "out")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}

	buildTarXZ(t, tarball, []tarEntry{
		{name: "../../escape.txt", mode: 0o644, typeflag: tar.TypeReg, body: []byte("pwn")},
	})

	// Native path must reject. The fallback (system tar) might not, so we
	// exercise the native path directly here to keep this test about the
	// containment guard rather than fallback behavior.
	// os.Root surfaces this as "path escapes from parent"; older string-
	// prefix code surfaced it as "illegal file path". Accept either so the
	// test pins the behavior (rejection) without coupling to wording.
	err := untarXZNative(tarball, out)
	if err == nil {
		t.Fatal("expected error for zip-slip entry, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "escapes from") && !strings.Contains(msg, "illegal file path") {
		t.Fatalf("expected path-traversal error, got: %v", err)
	}

	// Verify nothing escaped the temp dir.
	if _, statErr := os.Stat(filepath.Join(dir, "..", "escape.txt")); statErr == nil {
		t.Fatal("file escaped the extraction root")
	}
}

// T5b: zip-slip via a malicious symlink target is rejected.
func TestUntarXZ_RejectsZipSlipSymlink(t *testing.T) {
	if !canSymlink(t) {
		t.Skip("environment cannot create symlinks")
	}
	dir := t.TempDir()
	tarball := filepath.Join(dir, "evil.tar.xz")
	out := filepath.Join(dir, "out")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}

	buildTarXZ(t, tarball, []tarEntry{
		{name: "link", mode: 0o777, typeflag: tar.TypeSymlink, linkname: "../../../etc/passwd"},
	})

	err := untarXZNative(tarball, out)
	if err == nil {
		t.Fatal("expected error for symlink escape, got nil")
	}
	if !strings.Contains(err.Error(), "symlink target") {
		t.Fatalf("expected symlink-escape error, got: %v", err)
	}
}

// T6: corrupt xz triggers fallback to the system tar backend.
func TestUntarXZ_FallbackOnCorruptStream(t *testing.T) {
	dir := t.TempDir()
	tarball := filepath.Join(dir, "corrupt.tar.xz")
	if err := os.WriteFile(tarball, []byte("not actually xz"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}

	calls := 0
	prev := systemXZBackend
	t.Cleanup(func() { systemXZBackend = prev })
	systemXZBackend = func(in, outDir string) error {
		calls++
		// Sentinel proving the fallback ran with the right args.
		return os.WriteFile(filepath.Join(outDir, "fallback.marker"), []byte("ok"), 0o644)
	}

	if err := untarXZ(tarball, out); err != nil {
		t.Fatalf("untarXZ: %v", err)
	}
	if calls != 1 {
		t.Fatalf("system backend call count = %d, want 1", calls)
	}
	if _, err := os.Stat(filepath.Join(out, "fallback.marker")); err != nil {
		t.Fatalf("fallback marker missing: %v", err)
	}
}

// T7: a successful native extraction does NOT call the system backend.
func TestUntarXZ_NoFallbackOnSuccess(t *testing.T) {
	dir := t.TempDir()
	tarball := filepath.Join(dir, "ok.tar.xz")
	out := filepath.Join(dir, "out")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}

	buildTarXZ(t, tarball, []tarEntry{
		{name: "f.txt", mode: 0o644, typeflag: tar.TypeReg, body: []byte("ok")},
	})

	calls := 0
	prev := systemXZBackend
	t.Cleanup(func() { systemXZBackend = prev })
	systemXZBackend = func(in, outDir string) error {
		calls++
		return nil
	}

	if err := untarXZ(tarball, out); err != nil {
		t.Fatalf("untarXZ: %v", err)
	}
	if calls != 0 {
		t.Fatalf("system backend was called %d times on a successful native run; want 0", calls)
	}
}

// T8: when both paths fail, the system path's error is what surfaces.
func TestUntarXZ_FallbackPropagatesError(t *testing.T) {
	dir := t.TempDir()
	tarball := filepath.Join(dir, "corrupt.tar.xz")
	if err := os.WriteFile(tarball, []byte("garbage"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}

	sentinel := errors.New("system tar boom")
	prev := systemXZBackend
	t.Cleanup(func() { systemXZBackend = prev })
	systemXZBackend = func(in, outDir string) error { return sentinel }

	err := untarXZ(tarball, out)
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected system-path error to surface, got: %v", err)
	}
}

// Sanity check: ulikunitz/xz NewReader satisfies io.Reader and walking
// archive/tar over it works as the implementation assumes. Pinned here
// so a future dep bump that changes the API surface flags loudly.
func TestUntarXZ_LibraryAssumption(t *testing.T) {
	var buf bytes.Buffer
	xzw, err := xz.NewWriter(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(xzw, "hello"); err != nil {
		t.Fatal(err)
	}
	if err := xzw.Close(); err != nil {
		t.Fatal(err)
	}

	r, err := xz.NewReader(&buf)
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello" {
		t.Fatalf("xz round-trip = %q, want %q", got, "hello")
	}
}
